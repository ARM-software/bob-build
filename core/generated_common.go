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

type generateCommon struct {
	moduleBase
	simpleOutputProducer
	headerProducer
	Properties struct {
		GenerateProps
		Features
		FlagArgsBuild Build `blueprint:"mutated"`
	}
}

// generateCommon supports:
// * feature-specific properties
// * module enabling/disabling
// * module splitting for targets
// * use of {{match_srcs}} on some properties
// * properties that require escaping
// * sharing properties from defaults via `flag_defaults` property
var _ Featurable = (*generateCommon)(nil)
var _ enableable = (*generateCommon)(nil)
var _ splittable = (*generateCommon)(nil)
var _ matchSourceInterface = (*generateCommon)(nil)
var _ propertyEscapeInterface = (*generateCommon)(nil)
var _ defaultable = (*generateCommon)(nil)

func (m *generateCommon) init(properties *configProperties, list ...interface{}) {
	m.Properties.Features.Init(properties, list...)
	m.Properties.FlagArgsBuild.Host.init(properties, CommonProps{}, BuildProps{})
	m.Properties.FlagArgsBuild.Target.init(properties, CommonProps{}, BuildProps{})
}

func (m *generateCommon) shortName() string {
	return m.Name()
}

func (m *generateCommon) altName() string {
	return m.Name()
}

func (m *generateCommon) altShortName() string {
	return m.shortName()
}

// Workaround for Golang not having a way of querying superclasses
func (m *generateCommon) getGenerateCommon() *generateCommon {
	return m
}

func (m *generateCommon) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.GenerateProps}
}

func (m *generateCommon) Features() *Features {
	return &m.Properties.Features
}

func (m *generateCommon) getTarget() TgtType {
	return m.Properties.Target
}

func (m *generateCommon) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *generateCommon) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, installDepTag)
}

func (m *generateCommon) supportedVariants() []TgtType {
	return []TgtType{m.Properties.Target}
}

func (m *generateCommon) disable() {
	*m.Properties.Enabled = false
}

func (m *generateCommon) setVariant(variant TgtType) {
	if variant != m.Properties.Target {
		utils.Die("Variant mismatch: %s != %s", variant, m.Properties.Target)
	}
}

func (m *generateCommon) getSplittableProps() *SplittableProps {
	return &m.Properties.FlagArgsBuild.SplittableProps
}

func (m *generateCommon) getEscapeProperties() []*[]string {
	return []*[]string{
		&m.Properties.FlagArgsBuild.Asflags,
		&m.Properties.FlagArgsBuild.Cflags,
		&m.Properties.FlagArgsBuild.Conlyflags,
		&m.Properties.FlagArgsBuild.Cxxflags,
		&m.Properties.FlagArgsBuild.Ldflags}
}

func (m *generateCommon) getLegacySourceProperties() *LegacySourceProps {
	return &m.Properties.GenerateProps.LegacySourceProps
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `generateCommon` these are:
//   - Args
//   - Cmd
func (m *generateCommon) getMatchSourcePropNames() []string {
	return []string{"Cmd", "Args"}
}

// Populate the output from inout structures that have already been
// filled out. Note, if output directories need to be referenced, then
// inouts should be updated before calling this function.
func (m *generateCommon) recordOutputsFromInout(inouts []inout) {
	for _, inout := range inouts {
		m.outs = append(m.outs, inout.out...)
		m.implicitOuts = append(m.implicitOuts, inout.implicitOuts...)
	}
}

func (m *generateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *generateCommon) getDepfile() (name string, depfile bool) {
	depfile = proptools.Bool(m.Properties.Depfile)
	if depfile {
		name = getDepfileName(m.Name())
		return
	}
	return "", depfile
}

func (m *generateCommon) getRspfile() (name string, rspfile bool) {
	rspfile = m.Properties.Rsp_content != nil
	if rspfile {
		name = getRspfileName(m.Name())
		return
	}
	return "", rspfile
}

func (m *generateCommon) defaultableProperties() []interface{} {
	return []interface{}{
		&m.Properties.FlagArgsBuild.CommonProps,
		&m.Properties.FlagArgsBuild.BuildProps,
	}
}

func (m *generateCommon) defaults() []string {
	return m.Properties.Flag_defaults
}

func (m *generateCommon) hostBinName(mctx blueprint.ModuleContext) (name string) {
	mctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return mctx.OtherModuleDependencyTag(dep) == hostToolBinTag
		},
		func(module blueprint.Module) {
			_, bin_ok := module.(*binary)
			_, genbin_ok := module.(*generateBinary)
			if bin_ok || genbin_ok {
				name = module.Name()
			} else {
				mctx.PropertyErrorf("host_bin", "%s is not a `bob_binary` nor `bob_generate_binary`", module.Name())
			}
		})

	return
}

// hostBinOuts returns the tool binary ('host_bin') together with its
// target type and shared library dependencies for a generator module.
// This is different from the "tool" in that it used to depend on
// a bob_binary module.
func (m *generateCommon) hostBinOuts(mctx blueprint.ModuleContext) (string, []string, TgtType) {
	// No host_bin provided
	if m.Properties.Host_bin == nil {
		return "", []string{}, tgtTypeUnknown
	}

	hostBinOut := ""
	hostBinSharedLibsDeps := []string{}
	hostBinTarget := tgtTypeUnknown
	hostBinFound := false

	mctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		depTag := mctx.OtherModuleDependencyTag(child)

		if parent == mctx.Module() && depTag == hostToolBinTag {
			var outputs []string
			hostBinFound = true

			if b, ok := child.(*binary); ok {
				outputs = b.outputs()
				hostBinTarget = b.getTarget()
			} else if gb, ok := child.(*generateBinary); ok {
				outputs = gb.outputs()
			} else {
				mctx.PropertyErrorf("host_bin", "%s is not a `bob_binary` nor `bob_generate_binary`", parent.Name())
				return false
			}

			if len(outputs) != 1 {
				mctx.OtherModuleErrorf(child, "outputs() returned %d outputs", len(outputs))
			} else {
				hostBinOut = outputs[0]
			}

			return true // keep visiting
		} else if parent != mctx.Module() && depTag == sharedDepTag {
			if l, ok := child.(*sharedLibrary); ok {
				hostBinSharedLibsDeps = append(hostBinSharedLibsDeps, l.outputs()...)
			}

			return true // keep visiting
		} else {
			return false // stop visiting
		}
	})

	if !hostBinFound {
		mctx.ModuleErrorf("Could not find module specified by `host_bin: %v`", m.Properties.Host_bin)
	}

	return hostBinOut, hostBinSharedLibsDeps, hostBinTarget
}

func (m *generateCommon) getArgs(ctx blueprint.ModuleContext) (string, map[string]string, []string, TgtType) {
	g := getBackend(ctx)

	tc := g.getToolchain(m.Properties.Target)
	arBinary, _ := tc.getArchiver()
	asBinary, astargetflags := tc.getAssembler()
	cc, cctargetflags := tc.getCCompiler()
	cxx, cxxtargetflags := tc.getCXXCompiler()
	linker := tc.getLinker().getTool()
	ldtargetflags := tc.getLinker().getFlags()
	ldlibs := tc.getLinker().getLibs()

	props := &m.Properties.FlagArgsBuild

	args := map[string]string{
		"ar":              arBinary,
		"as":              asBinary,
		"asflags":         utils.Join(astargetflags, props.Asflags),
		"bob_config":      configFile,
		"bob_config_json": configJSONFile,
		"bob_config_opts": configOpts,
		"cc":              cc,
		"cflags":          strings.Join(props.Cflags, " "),
		"conlyflags":      strings.Join(append(cctargetflags, props.Conlyflags...), " "),
		"cxx":             cxx,
		"cxxflags":        strings.Join(append(cxxtargetflags, props.Cxxflags...), " "),
		"ldflags":         utils.Join(ldtargetflags, props.Ldflags),
		"ldlibs":          utils.Join(ldlibs, props.Ldlibs),
		"linker":          linker,
		"gen_dir":         m.outputDir(),
		"module_dir":      getBackendPathInSourceDir(g, ctx.ModuleDir()),
		"shared_libs_dir": g.sharedLibsDir(m.Properties.GenerateProps.Target),
		"src_dir":         g.sourceDir(),
	}

	args["build_wrapper"], _ = props.getBuildWrapperAndDeps(ctx)

	dependents := getDependentArgsAndFiles(ctx, args)

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
	cmd, toolArgs, dependentTools := m.processCmdTools(ctx, cmd)

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

func (m *generateCommon) processCmdTools(ctx blueprint.ModuleContext, cmd string) (string, map[string]string, []string) {

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
		g := getBackend(ctx)
		for _, tool := range m.Properties.Tools {
			toolPath := getBackendPathInSourceDir(g, tool)
			addToolsLabel(tool, toolPath)
			dependentTools = append(dependentTools, toolPath)
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

		label := filepath.Join(projectModuleDir(ctx), submatch[1])

		if toolPath, ok := toolsLabels[label]; ok {
			toolKey := "tool_" + strconv.Itoa(idx)
			cmd = strings.Replace(cmd, match, "${"+toolKey+"}", -1)
			args[toolKey] = toolPath
			idx++
		} else {
			ctx.ModuleErrorf("unknown tool %q in tools.", submatch[1])
		}
	}

	return cmd, args, dependentTools
}

func (m *generateCommon) getSourcesResolved(ctx blueprint.BaseModuleContext) []string {
	return m.Properties.getSourcesResolved(ctx)
}

func (m *generateCommon) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.LegacySourceProps.processPaths(ctx, g)
	m.Properties.InstallableProps.processPaths(ctx, g)

	if len(m.Properties.Tools) > 0 {
		toolPaths := []string{}
		for _, tool := range m.Properties.Tools {
			toolPaths = append(toolPaths, filepath.Join(projectModuleDir(ctx), tool))
		}
		m.Properties.Tools = toolPaths
	}
}

func (m *generateCommon) getAliasList() []string {
	return m.Properties.getAliasList()
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getGenerateCommonInterface interface {
	getGenerateCommon() *generateCommon
}

func getGenerateCommon(i interface{}) (*generateCommon, bool) {
	var gsc *generateCommon
	gsd, ok := i.(getGenerateCommonInterface)
	if ok {
		gsc = gsd.getGenerateCommon()
	}
	return gsc, ok
}