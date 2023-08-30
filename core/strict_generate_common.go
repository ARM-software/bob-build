package core

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

var variableRegex = regexp.MustCompile(`\$\(([A-Za-z- \._:0-9]+)\)`)
var locationRegex = regexp.MustCompile(`\$\{location ([a-zA-Z0-9\/\.:_-]+)\}`)

type ModuleStrictGenerateCommon struct {
	module.ModuleBase
	Properties struct {
		EnableableProps
		TagableProps
		Features
		StrictGenerateProps
	}
	deps []string
}

type StrictGenerateCommonInterface interface {
	pathProcessor
	file.Consumer
	file.Resolver
	Tagable
}

var _ StrictGenerateCommonInterface = (*ModuleStrictGenerateCommon)(nil)

func (m *ModuleStrictGenerateCommon) init(properties *config.Properties, list ...interface{}) {
	m.Properties.Features.Init(properties, list...)
}

func (m *ModuleStrictGenerateCommon) processPaths(ctx blueprint.BaseModuleContext) {
	m.deps = utils.MixedListToBobTargets(m.Properties.StrictGenerateProps.Tool_files)
	m.Properties.StrictGenerateProps.processPaths(ctx)
}

func (m *ModuleStrictGenerateCommon) GetTargets() []string {
	return m.Properties.GetTargets()
}

func (m *ModuleStrictGenerateCommon) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.GetFiles(ctx)
}

func (m *ModuleStrictGenerateCommon) GetDirectFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleStrictGenerateCommon) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleStrictGenerateCommon) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleStrictGenerateCommon) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.EnableableProps,
		&m.Properties.StrictGenerateProps,
		&m.Properties.TagableProps}
}

func (m *ModuleStrictGenerateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *ModuleStrictGenerateCommon) getArgs(ctx blueprint.ModuleContext) (string, map[string]string, []string, string) {
	hostLdLibraryPath := ""
	env := config.GetEnvironmentVariables()

	args := map[string]string{
		"bob_config":      env.ConfigFile,
		"bob_config_json": env.ConfigJSON,
		"bob_config_opts": env.ConfigOpts,
		"genDir":          backend.Get().SourceOutputDir(ctx.Module()),
	}

	dependents, fullDeps := getDependentArgsAndFiles(ctx, args)

	var hostBinName *string = nil

	if len(m.Properties.Tools) > 0 {
		// TODO: Currently limited to one tool
		hostBinName = &m.Properties.Tools[0]
	}

	hostBin, hostBinSharedLibs, hostTarget := hostBinOuts(hostBinName, ctx)
	if hostBin != "" {
		args["host_bin"] = hostBin
		dependents = append(dependents, hostBin)
		dependents = append(dependents, hostBinSharedLibs...)
		hostLdLibraryPath = "LD_LIBRARY_PATH=" + backend.Get().SharedLibsDir(hostTarget) + ":$$LD_LIBRARY_PATH "
	}

	cmd, toolArgs, dependentTools := m.processCmd(ctx, fullDeps)

	for k, v := range toolArgs {
		args[k] = v
	}

	dependents = append(dependents, dependentTools...)

	utils.StripUnusedArgs(args, cmd)

	return cmd, args, dependents, hostLdLibraryPath
}

func (m *ModuleStrictGenerateCommon) processCmd(ctx blueprint.ModuleContext, fullDeps map[string][]string) (string, map[string]string, []string) {
	cmd := m.preprocessCmd()

	dependentTools := []string{}
	toolsLabels := map[string]string{}
	args := map[string]string{}
	firstTool := ""

	addToolsLabel := func(label string, tool string) {
		if firstTool == "" {
			firstTool = label
		}
		if _, exists := toolsLabels[label]; !exists {
			toolsLabels[label] = tool
		} else {
			ctx.ModuleErrorf("multiple locations for label %q: %q and %q (do you have duplicate tools entries?)",
				label, toolsLabels[label], tool)
		}
	}

	if len(m.Properties.Tool_files) > 0 {
		for _, tool := range m.Properties.Tool_files {
			// If tool comes from other module with `:` notation
			// just fill up `toolsLabels` to not duplicate
			// `dependentTools` which has been already added by
			// `tag.GeneratedTag` dependencies.
			toolPath := ""
			if tool[0] == ':' {
				for modName, deps := range fullDeps {
					if modName == tool[1:] {
						// Grab all the outputs,genDir
						// those will be packed in one
						// `tool_x` in command
						toolPath = strings.Join(deps, " ")
						break
					}
				}

			} else {
				toolPath = getBackendPathInSourceDir(getGenerator(ctx), tool)
				dependentTools = append(dependentTools, toolPath)
			}
			addToolsLabel(tool, toolPath)
		}
	}

	r := regexp.MustCompile(`\$\{location ([^{}$]+)\}`)

	matches := r.FindAllString(cmd, -1)
	var idx = 1

	for _, match := range matches {
		submatch := r.FindStringSubmatch(match)
		label := submatch[1]

		if toolPath, ok := toolsLabels[label]; ok {
			toolKey := "tool_" + strconv.Itoa(idx)
			cmd = strings.Replace(cmd, match, "${"+toolKey+"}", -1)
			args[toolKey] = toolPath
			idx++
		} else {
			ctx.ModuleErrorf("unknown tool '%q' in tools in cmd:'%q', possible tools:'%q'.",
				label,
				cmd,
				toolsLabels)
		}
	}

	return cmd, args, dependentTools
}

func (m *ModuleStrictGenerateCommon) preprocessCmd() string {
	// Bob handles multiple tool files identically to android. e.g.
	// $(location tool2) == ${tool tool2}
	// However, android differs as it also allows you to use the tag to depend
	// on a tool created by a source dependency. Bob does this with special wildcards e.g.
	// $(location dependency) == ${dependency_out}
	// We must convert these correctly for the proxy object.

	cmd := *m.Properties.Cmd

	// Set default tool
	if strings.Contains(cmd, "${location}") {
		if len(m.Properties.Tools) > 0 {
			cmd = strings.Replace(cmd, "${location}", "${location "+m.Properties.Tools[0]+"}", -1)
		} else if len(m.Properties.Tool_files) > 0 {
			cmd = strings.Replace(cmd, "${location}", "${location "+m.Properties.Tool_files[0]+"}", -1)
		}
	}

	// Extract each substring that is a 'location <tag>'
	matches := locationRegex.FindAllStringSubmatch(cmd, -1)

	for _, v := range matches {
		tag := v[1]

		// If the tag refers to `tool_files` just continue
		if utils.Contains(m.Properties.Tool_files, tag) {
			continue
		}

		// Tag is a dependency
		if tag[0] == ':' {
			newString := strings.TrimPrefix(tag, ":")
			newString = "${" + newString + "_out}"
			cmd = strings.Replace(cmd, v[0], newString, 1)
			continue
		}

		if utils.Contains(m.Properties.Tools, tag) {
			cmd = strings.Replace(cmd, v[0], "${host_bin}", 1)
		} else {
			cmd = strings.Replace(cmd, v[0], "${"+tag+"_out}", 1)
		}
	}

	// Ninja reserves the `$(out)` property, but Bob needs it to contain all
	// outputs, not just explicit ones. So replace that too.
	cmd = strings.Replace(cmd, "${out}", "${_out_}", -1)

	return cmd
}

func (m *ModuleStrictGenerateCommon) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleStrictGenerateCommon) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleStrictGenerateCommon) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleStrictGenerateCommon) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

func (m *ModuleStrictGenerateCommon) GenerateBuildActions(blueprint.ModuleContext) {
	// Stub to fullfill blueprint.Module
}

// Module implementing `StrictGenerator`
// are able to generate output files
type StrictGenerator interface {
	getStrictGenerateCommon() *ModuleStrictGenerateCommon
}

func getStrictGenerateCommon(i interface{}) (*ModuleStrictGenerateCommon, bool) {
	var m *ModuleStrictGenerateCommon
	sg, ok := i.(StrictGenerator)
	if ok {
		m = sg.getStrictGenerateCommon()
	}
	return m, ok
}
