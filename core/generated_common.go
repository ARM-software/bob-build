/*
 * Copyright 2023 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// For bob_transform_source each src in the glob will get its own
// ninja rule. Each src can have multiple outputs.
//
// To support this the inout type is used to group the outputs
// associated with each src.
//
// Inputs should be specified relative to the working directory, to allow both
// "normal" and generated inputs to be used.
type inout struct {
	in           []string
	out          []string
	depfile      string
	implicitSrcs []string
	implicitOuts []string
	rspfile      string
}

// Add a prefix to all output paths
func prefixInoutsWithOutputDir(inouts []inout, dir string) {
	for i := range inouts {
		inouts[i].out = utils.PrefixDirs(inouts[i].out, dir)
		inouts[i].implicitOuts = utils.PrefixDirs(inouts[i].implicitOuts, dir)
		if inouts[i].depfile != "" {
			inouts[i].depfile = filepath.Join(dir, inouts[i].depfile)
		}
		if inouts[i].rspfile != "" {
			inouts[i].rspfile = filepath.Join(dir, inouts[i].rspfile)
		}
	}
}

// Modules implementing hostBin are able to supply a host binary that can be executed
type hostBin interface {
	hostBin() string
}

// When referencing libraries provided by a generator module use "module/path/to/lib"
// This function splits the reference into the module and the library.
func splitGeneratedComponent(comp string) (module string, lib string) {
	split := strings.Split(comp, "/")

	if len(split) < 2 {
		utils.Die("Generated component %s does not specify module and lib", comp)
	}

	return split[0], strings.Join(split[1:], "/")
}

type ModuleGenerateCommon struct {
	module.ModuleBase
	simpleOutputProducer
	headerProducer
	Properties struct {
		GenerateProps
		Features
		FlagArgsBuild Build `blueprint:"mutated"`
	}
	deps []string
}

// ModuleGenerateCommon supports:
// * feature-specific properties
// * module enabling/disabling
// * module splitting for targets
// * use of {{match_srcs}} on some properties
// * properties that require escaping
// * sharing properties from defaults via `flag_defaults` property
var _ Featurable = (*ModuleGenerateCommon)(nil)
var _ enableable = (*ModuleGenerateCommon)(nil)
var _ splittable = (*ModuleGenerateCommon)(nil)
var _ matchSourceInterface = (*ModuleGenerateCommon)(nil)
var _ propertyEscapeInterface = (*ModuleGenerateCommon)(nil)
var _ defaultable = (*ModuleGenerateCommon)(nil)

func (m *ModuleGenerateCommon) implicitOutputs() []string {
	return []string{}
}

func (m *ModuleGenerateCommon) outputs() []string {
	return m.outs
}

func (m *ModuleGenerateCommon) init(properties *config.Properties, list ...interface{}) {
	m.Properties.Features.Init(properties, list...)
	m.Properties.FlagArgsBuild.Host.init(properties, CommonProps{}, BuildProps{})
	m.Properties.FlagArgsBuild.Target.init(properties, CommonProps{}, BuildProps{})
}

func (m *ModuleGenerateCommon) shortName() string {
	return m.Name()
}

func (m *ModuleGenerateCommon) altName() string {
	return m.Name()
}

func (m *ModuleGenerateCommon) altShortName() string {
	return m.shortName()
}

// Workaround for Golang not having a way of querying superclasses
func (m *ModuleGenerateCommon) getGenerateCommon() *ModuleGenerateCommon {
	return m
}

func (m *ModuleGenerateCommon) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.GenerateProps}
}

func (m *ModuleGenerateCommon) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleGenerateCommon) getTarget() toolchain.TgtType {
	return m.Properties.Target
}

func (m *ModuleGenerateCommon) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *ModuleGenerateCommon) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, InstallTag)
}

func (m *ModuleGenerateCommon) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{m.Properties.Target}
}

func (m *ModuleGenerateCommon) disable() {
	*m.Properties.Enabled = false
}

func (m *ModuleGenerateCommon) setVariant(variant toolchain.TgtType) {
	if variant != m.Properties.Target {
		utils.Die("Variant mismatch: %s != %s", variant, m.Properties.Target)
	}
}

func (m *ModuleGenerateCommon) getSplittableProps() *SplittableProps {
	return &m.Properties.FlagArgsBuild.SplittableProps
}

func (m *ModuleGenerateCommon) getEscapeProperties() []*[]string {
	return []*[]string{
		&m.Properties.FlagArgsBuild.Asflags,
		&m.Properties.FlagArgsBuild.Cflags,
		&m.Properties.FlagArgsBuild.Conlyflags,
		&m.Properties.FlagArgsBuild.Cxxflags,
		&m.Properties.FlagArgsBuild.Ldflags}
}

func (m *ModuleGenerateCommon) getLegacySourceProperties() *LegacySourceProps {
	return &m.Properties.GenerateProps.LegacySourceProps
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `generateCommon` these are:
//   - Args
//   - Cmd
func (m *ModuleGenerateCommon) getMatchSourcePropNames() []string {
	return []string{"Cmd", "Args"}
}

// Populate the output from inout structures that have already been
// filled out. Note, if output directories need to be referenced, then
// inouts should be updated before calling this function.
func (m *ModuleGenerateCommon) recordOutputsFromInout(inouts []inout) {
	for _, inout := range inouts {
		m.outs = append(m.outs, inout.out...)
	}
}

func (m *ModuleGenerateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *ModuleGenerateCommon) getDepfile() (name string, depfile bool) {
	depfile = proptools.Bool(m.Properties.Depfile)
	if depfile {
		name = getDepfileName(m.Name())
		return
	}
	return "", depfile
}

func (m *ModuleGenerateCommon) getRspfile() (name string, rspfile bool) {
	rspfile = m.Properties.Rsp_content != nil
	if rspfile {
		name = getRspfileName(m.Name())
		return
	}
	return "", rspfile
}

func (m *ModuleGenerateCommon) defaultableProperties() []interface{} {
	return []interface{}{
		&m.Properties.FlagArgsBuild.CommonProps,
		&m.Properties.FlagArgsBuild.BuildProps,
	}
}

func (m *ModuleGenerateCommon) defaults() []string {
	return m.Properties.Flag_defaults
}

func (m *ModuleGenerateCommon) hostBinName(ctx blueprint.ModuleContext) (name string) {
	ctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(dep) == HostToolBinaryTag
		},
		func(module blueprint.Module) {
			_, bin_ok := module.(*ModuleBinary)
			_, genbin_ok := module.(*generateBinary)
			if bin_ok || genbin_ok {
				name = module.Name()
			} else {
				ctx.PropertyErrorf("host_bin", "%s is not a `bob_binary` nor `bob_generate_binary`", module.Name())
			}
		})

	return
}

// hostBinOuts returns the tool binary ('host_bin') together with its
// target type and shared library dependencies for a generator module.
// This is different from the "tool" in that it used to depend on
// a bob_binary module.
func (m *ModuleGenerateCommon) hostBinOuts(ctx blueprint.ModuleContext) (string, []string, toolchain.TgtType) {
	// No host_bin provided
	if m.Properties.Host_bin == nil {
		return "", []string{}, toolchain.TgtTypeUnknown
	}

	hostBinOut := ""
	hostBinSharedLibsDeps := []string{}
	hostBinTarget := toolchain.TgtTypeUnknown
	hostBinFound := false

	ctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		depTag := ctx.OtherModuleDependencyTag(child)

		if parent == ctx.Module() && depTag == HostToolBinaryTag {
			var outputs []string
			hostBinFound = true

			if b, ok := child.(*ModuleBinary); ok {
				outputs = b.outputs()
				hostBinTarget = b.getTarget()
			} else if gb, ok := child.(*generateBinary); ok {
				outputs = gb.outputs()
			} else {
				ctx.PropertyErrorf("host_bin", "%s is not a `bob_binary` nor `bob_generate_binary`", parent.Name())
				return false
			}

			if len(outputs) != 1 {
				ctx.OtherModuleErrorf(child, "outputs() returned %d outputs", len(outputs))
			} else {
				hostBinOut = outputs[0]
			}

			return true // keep visiting
		} else if parent != ctx.Module() && depTag == SharedTag {
			if l, ok := child.(*ModuleSharedLibrary); ok {
				hostBinSharedLibsDeps = append(hostBinSharedLibsDeps, l.outputs()...)
			}

			return true // keep visiting
		} else {
			return false // stop visiting
		}
	})

	if !hostBinFound {
		ctx.ModuleErrorf("Could not find module specified by `host_bin: %v`", m.Properties.Host_bin)
	}

	return hostBinOut, hostBinSharedLibsDeps, hostBinTarget
}

func (m *ModuleGenerateCommon) getArgs(ctx blueprint.ModuleContext) (string, map[string]string, []string, toolchain.TgtType) {
	b := backend.Get()

	tc := b.GetToolchain(m.Properties.Target)
	arBinary, _ := tc.GetArchiver()
	asBinary, astargetflags := tc.GetAssembler()
	cc, cctargetflags := tc.GetCCompiler()
	cxx, cxxtargetflags := tc.GetCXXCompiler()
	linker := tc.GetLinker().GetTool()
	ldtargetflags := tc.GetLinker().GetFlags()
	ldlibs := tc.GetLinker().GetLibs()

	props := &m.Properties.FlagArgsBuild

	env := config.GetEnvironmentVariables()
	args := map[string]string{
		"ar":              arBinary,
		"as":              asBinary,
		"asflags":         utils.Join(astargetflags, props.Asflags),
		"bob_config":      env.ConfigFile,
		"bob_config_json": env.ConfigJSON,
		"bob_config_opts": env.ConfigOpts,
		"cc":              cc,
		"cflags":          strings.Join(props.Cflags, " "),
		"conlyflags":      strings.Join(append(cctargetflags, props.Conlyflags...), " "),
		"cxx":             cxx,
		"cxxflags":        strings.Join(append(cxxtargetflags, props.Cxxflags...), " "),
		"ldflags":         utils.Join(ldtargetflags, props.Ldflags),
		"ldlibs":          utils.Join(ldlibs, props.Ldlibs),
		"linker":          linker,
		"gen_dir":         backend.Get().SourceOutputDir(ctx.Module()),
		"module_dir":      getBackendPathInSourceDir(getGenerator(ctx), ctx.ModuleDir()),
		"shared_libs_dir": b.SharedLibsDir(m.Properties.GenerateProps.Target),
		"src_dir":         b.SourceDir(),
	}

	args["build_wrapper"], _ = props.GetBuildWrapperAndDeps(ctx)

	dependents, fullDeps := getDependentArgsAndFiles(ctx, args)

	hostBin, hostBinSharedLibs, hostTarget := m.hostBinOuts(ctx)
	if hostBin != "" {
		args["host_bin"] = hostBin
		dependents = append(dependents, hostBin)
		dependents = append(dependents, hostBinSharedLibs...)
	}

	// Args can contain other parameters, so replace that immediately
	cmd := strings.Replace(proptools.String(m.Properties.Cmd), "${args}", strings.Join(m.Properties.Args, " "), -1)
	// Ninja reserves the `${out}` property, but Bob needs it to contain all
	// outputs, not just explicit ones. So replace that too.
	cmd = strings.Replace(cmd, "${out}", "${_out_}", -1)
	cmd, toolArgs, dependentTools := m.processCmdTools(ctx, cmd, fullDeps)

	for k, v := range toolArgs {
		args[k] = v
	}

	dependents = append(dependents, dependentTools...)

	if proptools.Bool(m.Properties.Depfile) && !utils.ContainsArg(cmd, "depfile") {
		utils.Die("%s depfile is true, but ${depfile} not used in cmd", m.Name())
	}
	if utils.ContainsArg(cmd, "bob_config") || utils.ContainsArg(cmd, "bob_config_json") {
		if !proptools.Bool(m.Properties.Depfile) {
			utils.Die("%s references Bob config but depfile not enabled. "+
				"Config dependencies must be declared via a depfile!", m.Name())
		}
	}

	return cmd, args, dependents, hostTarget
}

func (m *ModuleGenerateCommon) processCmdTools(ctx blueprint.ModuleContext, cmd string, fullDeps map[string][]string) (string, map[string]string, []string) {

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

	if len(m.Properties.Tools) > 0 {
		for _, tool := range m.Properties.Tools {
			// If tool comes from other module with `:` notation
			// just fill up `toolsLabels` to not duplicate
			// `dependentTools` which has been already added by
			// `GeneratedTag` dependencies.
			toolPath := ""
			if tool[0] == ':' {
				for modName, deps := range fullDeps {
					if modName == tool[1:] {
						// Grab all the outputs,
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

	// add first tool for ${tool}
	if utils.ContainsArg(cmd, "tool") {
		args["tool"] = toolsLabels[firstTool]
	}

	r := regexp.MustCompile(`\${tool ([^{}]+)}`)

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

var toolTagRegex = regexp.MustCompile(`\$\{tool ([a-zA-Z0-9\/\.:_-]+)\}`)

func (m *ModuleGenerateCommon) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.LegacySourceProps.processPaths(ctx)
	m.Properties.InstallableProps.processPaths(ctx)

	if len(m.Properties.Tools) > 0 {
		m.deps = utils.MixedListToBobTargets(m.Properties.Tools)
		tools_targets := utils.PrefixAll(m.deps, ":")
		m.Properties.Tools = utils.PrefixDirs(utils.MixedListToFiles(m.Properties.Tools), projectModuleDir(ctx))
		m.Properties.Tools = append(m.Properties.Tools, tools_targets...)
	}

	prefix := projectModuleDir(ctx)

	// TODO: add this test case
	if m.Properties.Cmd != nil {
		matches := toolTagRegex.FindAllStringSubmatch(*m.Properties.Cmd, -1)
		for _, v := range matches {
			tag := v[1]
			if tag[0] == ':' {
				continue
			}
			newTag := utils.PrefixDirs([]string{tag}, prefix)[0]
			// Replacing with space allows us to not replace the same basename more than once if it appears
			// multiple times.
			newCmd := strings.Replace(*m.Properties.Cmd, " "+tag, " "+newTag, -1)
			m.Properties.Cmd = &newCmd
		}
	}

}

func (m *ModuleGenerateCommon) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.LegacySourceProps.ResolveFiles(ctx)
}

func (m *ModuleGenerateCommon) getAliasList() []string {
	return m.Properties.getAliasList()
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getGenerateCommonInterface interface {
	getGenerateCommon() *ModuleGenerateCommon
}

func getGenerateCommon(i interface{}) (*ModuleGenerateCommon, bool) {
	var gsc *ModuleGenerateCommon
	gsd, ok := i.(getGenerateCommonInterface)
	if ok {
		gsc = gsd.getGenerateCommon()
	}
	return gsc, ok
}
