/*
 * Copyright 2018-2019 Arm Limited.
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
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/abstr"
	"github.com/ARM-software/bob-build/utils"
)

var (
	generatedHeaderTag = dependencyTag{name: "generated_headers"}
	generatedSourceTag = dependencyTag{name: "generated_sources"}
	generatedDepTag    = dependencyTag{name: "generated_dep"}
	encapsulatesTag    = dependencyTag{name: "source_encapsulation"}
	hostToolBinTag     = dependencyTag{name: "host_tool_bin"}
)

// For bob_transform_source each src in the glob will get its own
// ninja rule. Each src can have multiple outputs.
//
// To support this the inout type is used to group the outputs
// associated with each src.
//
// Inputs are relative to the project directory (they include module
// directory but not base source directory). Generated inputs are
// therefore kept separately in gen_in.
type inout struct {
	srcIn        []string
	genIn        []string
	out          []string
	depfile      string
	implicitSrcs []string
	implicitOuts []string
}

// GenerateProps contains the module properties that allow generation of
// output from arbitrary commands
type GenerateProps struct {
	SourceProps
	AliasableProps
	EnableableProps
	InstallableProps

	/* The command that is to be run for this source generation.
	 * Substitutions can be made in the command, by using $name_of_var. A list of substitutions that can be used:
	 * $gen_dir    - the path to the directory which belongs to this source generator
	 * $in         - the path to the sources - space-delimited
	 * $out        - the path to the targets - space-delimited
	 * $args       - the value of "args" - space-delimited
	 * $tool       - the path to the tool
	 * $host_bin   - the path to the binary that is produced by the host_bin module
	 * $(name)_dir - the build directory for each dep in generated_dep
	 * $src_dir    - the path to the project source directory - this will be different than the build source directory
	 *               for Android.
	 * $module_dir - the path to the module directory */
	Cmd string

	// A path to the tool that is to be used in cmd. If $tool is in the command variable, then this will be replaced.
	// with the path to this tool
	Tool string

	// Adds a dependency on a binary with `host_supported: true` which is used by this module.
	// The path can be referenced in cmd as ${host_bin}.
	Host_bin string

	// Values to use on Android for LOCAL_MODULE_TAGS, defining which builds this module is built for
	// TODO: Hide this in Android-specific properties
	Tags []string

	// A list of other modules that this generator depends on. The dependencies can be used in the command through
	// $name_of_dependency_dir .
	Module_deps []string

	// A list of other modules that this generator depends on. The dependencies will be add to the list of srcs
	Module_srcs []string

	// A list of args that will be spaceseparated and add to the cmd
	Args []string

	// Used to indicate that the console should be used.
	Console bool

	// A list of source modules that this bob_generated_source will encapsulate.
	// When this module is used with generated_headers, the named modules' export_gen_include_dirs will be forwarded.
	// When this module is used with generated_sources, the named modules' outputs will be supplied as sources.
	Encapsulates []string

	// Additional include paths to add for modules that use generate_headers.
	// This will be defined relative to the module-specific build directory
	Export_gen_include_dirs []string

	// The defaults used to retrieve cflags
	Flag_defaults []string

	// The target type - must be either "host" or "target"
	Target tgtType
}

type generateCommon struct {
	moduleBase
	Properties struct {
		GenerateProps
		Features
		FlagArgsBuild Build `blueprint:"mutated"`
	}
}

// Modules implementing hostBin are able to supply a host binary that can be executed
type hostBin interface {
	hostBin() string
}

// Modules implementing generatedDepender can depend on any of generator
// modules (bob_generate_source, bob_transform_source,
// bob_generate_static_lib, bob_generate_shared_lib, bob_generated_binary)
type generatedDepender interface {
	build() *Build
}

// When referencing libraries provided by a generator module use "module/path/to/lib"
// This function splits the reference into the module and the library.
func splitGeneratedComponent(comp string) (module string, lib string) {
	split := strings.Split(comp, "/")

	if len(split) < 2 {
		panic(errors.New("Generated component " + comp + " does not specify module and lib"))
	}

	return split[0], strings.Join(split[1:], "/")
}

func (m *generateCommon) outputDir(g generatorBackend) string {
	return g.sourceOutputDir(m)
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

func (m *generateCommon) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.GenerateProps}
}

func (m *generateCommon) features() *Features {
	return &m.Properties.Features
}

func (m *generateCommon) getTarget() tgtType {
	return m.Properties.Target
}

func (m *generateCommon) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *generateCommon) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, installDepTag)
}

func (m *generateCommon) supportedVariants() []tgtType {
	return []tgtType{m.Properties.Target}
}

func (m *generateCommon) disable() {
	*m.Properties.Enabled = false
}

func (m *generateCommon) setVariant(variant tgtType) {
	if variant != m.Properties.Target {
		panic(fmt.Errorf("Variant mismatch: %s != %s", variant, m.Properties.Target))
	}
}

func (m *generateCommon) getSplittableProps() *SplittableProps {
	return &m.Properties.FlagArgsBuild.SplittableProps
}

// Return a list of headers generated by this module with full paths
func getHeadersGenerated(g generatorBackend, m dependentInterface) []string {
	return utils.Filter(utils.IsHeader, m.outputs(g), m.implicitOutputs(g))
}

// Return a list of source files (not headers) generated by this module with full paths
func getSourcesGenerated(g generatorBackend, m dependentInterface) []string {
	return utils.Filter(utils.IsNotHeader, m.outputs(g), m.implicitOutputs(g))
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

func (m *generateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

// GenerateSourceProps are properties of 'bob_generate_source', i.e. a module
// type which can generate sources using a single execution
// The command will be run once - with $in being the paths in "srcs" and $out being the paths in "out".
// The working directory will be the source directory, and all paths will be relative to the source directory
// if not else noted.
type GenerateSourceProps struct {
	// The list of files that will be output.
	Out []string
	// File generated by the command describing discovered dependencies.
	Depfile string
	// List of implicit sources. Implicit sources are input files that do not get mentioned on the command line,
	// and are not specified in the explicit sources.
	Implicit_srcs []string
	// List of implicit outputs. Implicit outputs are output files that do not get mentioned on the command line.
	Implicit_outs []string
}

type generateSource struct {
	generateCommon
	Properties struct {
		GenerateSourceProps
	}
}

func (m *generateSource) outputs(g generatorBackend) []string {
	return utils.PrefixDirs(m.Properties.Out, m.outputDir(g))
}

func (m *generateSource) implicitOutputs(g generatorBackend) []string {
	return utils.PrefixDirs(m.Properties.Implicit_outs, m.outputDir(g))
}

func (m *generateSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		inouts := m.Inouts(ctx, g)
		g.generateSourceActions(m, ctx, inouts)
	}
}

func (m *generateSource) topLevelProperties() []interface{} {
	return append(m.generateCommon.topLevelProperties(), &m.Properties.GenerateSourceProps)
}

// Returns the tool binary for a generateSource module. This is different from the "tool"
// in that it used to depend on a bob_binary module
func (m *generateCommon) getHostBin(ctx blueprint.ModuleContext) (string, tgtType) {
	g := getBackend(ctx)
	toolBin := ""
	toolTarget := tgtTypeUnknown

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(m) == hostToolBinTag
		},
		func(m blueprint.Module) {
			if b, ok := m.(*binary); ok {
				toolBin = b.outputs(g)[0]
				toolTarget = b.getTarget()
			} else if b, ok := m.(*generateBinary); ok {
				toolBin = b.outputs(g)[0]
			} else {
				panic(errors.New("bob_generate_source must depend on a binary, not a library"))
			}
		})
	return toolBin, toolTarget
}

// Returns the dependents of a generateSource module. This is used for more complex dependencies, where
// the dependencies are not just a binary or a headers, but where the paths are used directly in a script
func getDependentArgsAndFiles(ctx blueprint.ModuleContext, args map[string]string) (depfiles []string) {
	g := getBackend(ctx)
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			depTag := ctx.OtherModuleDependencyTag(m)
			return depTag == generatedDepTag || depTag == encapsulatesTag
		},
		func(m blueprint.Module) {
			gen, ok := m.(dependentInterface)
			if !ok {
				panic(errors.New(reflect.TypeOf(m).String() + " is not a valid dependent interface"))
			}

			depName := ctx.OtherModuleName(m)
			args[depName+"_dir"] = gen.outputDir(g)
			args[depName+"_out"] = strings.Join(gen.outputs(g), " ")
			depfiles = append(depfiles, gen.outputs(g)...)
			depfiles = append(depfiles, gen.implicitOutputs(g)...)
		})

	return
}

func (m *generateCommon) getArgs(ctx blueprint.ModuleContext) (string, map[string]string, []string, tgtType) {
	g := getBackend(ctx)

	tc := g.getToolchain(m.Properties.Target)
	arBinary, _ := tc.getArchiver()
	asBinary, astargetflags := tc.getAssembler()
	cc, cctargetflags := tc.getCCompiler()
	cxx, cxxtargetflags := tc.getCXXCompiler()
	linker, ldflags := tc.getLinker()

	props := &m.Properties.FlagArgsBuild

	args := map[string]string{
		"ar":                arBinary,
		"as":                asBinary,
		"asflags":           utils.Join(astargetflags, props.Asflags),
		"bob_config":        filepath.Join(g.buildDir(), configName),
		"bob_config_opts":   configOpts,
		"cc":                cc,
		"cflags":            strings.Join(props.Cflags, " "),
		"conlyflags":        strings.Join(append(cctargetflags, props.Conlyflags...), " "),
		"cxx":               cxx,
		"cxxflags":          strings.Join(append(cxxtargetflags, props.Cxxflags...), " "),
		"ldflags":           utils.Join(ldflags, props.Ldflags),
		"linker":            linker,
		"gen_dir":           g.sourceOutputDir(m),
		"headers_generated": "",
		"module_dir":        filepath.Join(g.sourcePrefix(), ctx.ModuleDir()),
		"shared_libs_dir":   g.sharedLibsDir(m.Properties.GenerateProps.Target),
		"src_dir":           g.sourcePrefix(),
		"srcs_generated":    "",
	}

	args["build_wrapper"], _ = props.getBuildWrapperAndDeps(ctx)

	dependents := getDependentArgsAndFiles(ctx, args)

	if m.Properties.Tool != "" {
		toolPath := filepath.Join(g.sourcePrefix(), ctx.ModuleDir(), m.Properties.Tool)
		args["tool"] = toolPath
		dependents = append(dependents, toolPath)
	}

	hostBin, hostTarget := m.getHostBin(ctx)
	if hostBin != "" {
		args["host_bin"] = hostBin
		dependents = append(dependents, hostBin)
	}

	// Args can contain other parameters, so replace that immediately
	cmd := strings.Replace(m.Properties.Cmd, "${args}", strings.Join(m.Properties.Args, " "), -1)

	return cmd, args, dependents, hostTarget
}

func (m *generateCommon) getSources(ctx abstr.ModuleContext) []string {
	return m.Properties.getSources(ctx)
}

func (m *generateCommon) processPaths(ctx abstr.ModuleContext, g generatorBackend) {
	m.Properties.SourceProps.processPaths(ctx, g)
	m.Properties.Export_gen_include_dirs = utils.PrefixDirs(m.Properties.Export_gen_include_dirs, g.sourceOutputDir(m))
}

func (m *generateCommon) getAliasList() []string {
	return m.Properties.getAliasList()
}

func (m *generateSource) Inouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var io inout
	io.srcIn = utils.PrefixDirs(m.getSources(ctx), g.sourcePrefix())
	io.genIn = utils.NewStringSlice(m.generateCommon.Properties.SourceProps.Specials, getGeneratedFiles(ctx, g))
	io.out = m.outputs(g)
	if m.Properties.Depfile != "" {
		io.depfile = filepath.Join(g.sourceOutputDir(&m.generateCommon), m.Properties.Depfile)
	}
	io.implicitSrcs = utils.PrefixDirs(m.Properties.Implicit_srcs, filepath.Join(g.sourcePrefix(), ctx.ModuleDir()))
	io.implicitOuts = m.implicitOutputs(g)

	return []inout{io}
}

func (m *generateSource) filesToInstall(ctx abstr.ModuleContext, g generatorBackend) []string {
	// Install everything that we generate
	return m.outputs(g)
}

// TransformSourceProps contains the properties allowed in the
// bob_transform_source module. This module supports one command execution
// per input file.
type TransformSourceProps struct {
	// The regular expression that is used to transform the source path to the target path.
	Out struct {
		// Regular expression to capture groups from srcs
		Match string
		// Names of outputs, which can use capture groups from match
		Replace []string
		// File generated by the command describing discovered dependencies, which can use capture groups from match
		Depfile string
		// List of implicit sources. Implicit sources are input files that do not get mentioned on the command line,
		// and are not specified in the explicit sources.
		Implicit_srcs []string
		// List of implicit outputs, which can use capture groups from match.
		// Implicit outputs are output files that do not get mentioned on the command line.
		Implicit_outs []string
	}
}

func (m *transformSource) outputs(g generatorBackend) []string {
	return m.outs
}

func (m *transformSource) implicitOutputs(g generatorBackend) []string {
	return m.implicitOuts
}

func (m *transformSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		inouts := m.Inouts(ctx, g)
		for _, inout := range inouts {
			m.outs = append(m.outs, inout.out...)
			m.implicitOuts = append(m.implicitOuts, inout.implicitOuts...)
		}
		g.transformSourceActions(m, ctx, inouts)
	}
}

func (m *transformSource) topLevelProperties() []interface{} {
	return append(m.generateCommon.topLevelProperties(), &m.Properties.TransformSourceProps)
}

func (m *transformSource) Inouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var inouts []inout
	re := regexp.MustCompile(m.Properties.Out.Match)

	// For a transform source every input file is expected to be
	// converted to an output file. So each file in srcs will get its
	// own inout and each generated input (from module_srcs) will also
	// get its own inout. That means one of src_in or gen_in is always
	// empty, and the other has a single element.
	empty := []string{}

	for _, src := range m.getSources(ctx) {
		ins := []string{filepath.Join(g.sourcePrefix(), src)}
		outs := []string{}
		depfile := ""
		implicitSrcs := []string{}
		implicitOuts := []string{}

		for _, rep := range m.Properties.Out.Replace {
			out := filepath.Join(g.sourceOutputDir(&m.generateCommon), re.ReplaceAllString(src, rep))
			outs = append(outs, out)
		}

		for _, implOut := range m.Properties.Out.Implicit_outs {
			implOut = filepath.Join(g.sourceOutputDir(&m.generateCommon), re.ReplaceAllString(src, implOut))
			implicitOuts = append(implicitOuts, implOut)
		}

		if m.Properties.Out.Depfile != "" {
			depfile = filepath.Join(g.sourceOutputDir(&m.generateCommon),
				re.ReplaceAllString(src, m.Properties.Out.Depfile))
		}

		for _, implSrc := range m.Properties.Out.Implicit_srcs {
			implSrc = re.ReplaceAllString(src, implSrc)
			implSrc = filepath.Join(g.sourcePrefix(), ctx.ModuleDir(), implSrc)
			implicitSrcs = append(implicitSrcs, implSrc)
		}

		inouts = append(inouts, inout{ins, empty, outs, depfile, implicitSrcs, implicitOuts})
	}
	for _, src := range utils.NewStringSlice(m.generateCommon.Properties.Specials, getGeneratedFiles(ctx, g)) {
		ins := []string{src}
		outs := []string{}
		depfile := ""
		implicitSrcs := []string{}
		implicitOuts := []string{}

		for _, rep := range m.Properties.Out.Replace {
			out := filepath.Join(g.sourceOutputDir(&m.generateCommon), re.ReplaceAllString(src, rep))
			outs = append(outs, out)
		}

		for _, implOut := range m.Properties.Out.Implicit_outs {
			implOut = filepath.Join(g.sourceOutputDir(&m.generateCommon), re.ReplaceAllString(src, implOut))
			implicitOuts = append(implicitOuts, implOut)
		}

		if m.Properties.Out.Depfile != "" {
			depfile = filepath.Join(g.sourceOutputDir(&m.generateCommon),
				re.ReplaceAllString(src, m.Properties.Out.Depfile))
		}

		for _, implSrc := range m.Properties.Out.Implicit_srcs {
			implSrc = re.ReplaceAllString(src, implSrc)
			implSrc = filepath.Join(g.sourcePrefix(), ctx.ModuleDir(), implSrc)
			implicitSrcs = append(implicitSrcs, implSrc)
		}

		inouts = append(inouts, inout{empty, ins, outs, depfile, implicitSrcs, implicitOuts})
	}

	return inouts
}

func (m *transformSource) filesToInstall(ctx abstr.ModuleContext, g generatorBackend) []string {
	// Install everything that we generate
	return m.outputs(g)
}

// The module that can generate sources using a multiple execution
// The command will be run once per src file- with $in being the path in "srcs" and $out being the path transformed
// through the regexp defined by out.match and out.replace. The regular expression that is used is
// in regexp.compiled(out.Match).ReplaceAllString(src[i], out.Replace). See https://golang.org/pkg/regexp/ for more
// information.
// The working directory will be the source directory, and all paths will be relative to the source directory
// if not else noted
type transformSource struct {
	generateCommon
	Properties struct {
		TransformSourceProps
	}
	outs         []string
	implicitOuts []string
}

func generateSourceFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &generateSource{}
	module.generateCommon.Properties.Features.Init(config.getAvailableFeatures(),
		GenerateProps{}, GenerateSourceProps{})
	return module, []interface{}{&module.generateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}

func transformSourceFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &transformSource{}
	module.generateCommon.Properties.Features.Init(config.getAvailableFeatures(),
		GenerateProps{}, TransformSourceProps{})
	return module, []interface{}{&module.generateCommon.Properties,
		&module.Properties,
		&module.SimpleName.Properties}
}

// ModuleContext Helpers

// Return the outputs() and implicitOutputs() of all GeneratedSource dependencies of
// the current module. The current module can be generated or a library, and the
// dependencies can be anything implementing DependentInterface (so "generated"
// is a misnomer, because this includes libraries, too).
func getGeneratedFiles(ctx blueprint.ModuleContext, g generatorBackend) []string {
	var srcs []string
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == generatedSourceTag },
		func(m blueprint.Module) {
			if gs, ok := m.(dependentInterface); ok {
				srcs = append(srcs, gs.outputs(g)...)
				srcs = append(srcs, gs.implicitOutputs(g)...)
			} else {
				panic(errors.New(ctx.OtherModuleName(m) + " does not have outputs"))
			}
		})
	return srcs
}

func generatedDependerMutator(mctx blueprint.BottomUpMutatorContext) {
	if e, ok := mctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	// Things which depend on generated/transformed sources
	if gd, ok := mctx.Module().(generatedDepender); ok {
		if _, ok := mctx.Module().(*defaults); ok {
			// We do not want to add dependencies for defaults
			return
		}
		b := gd.build()
		mctx.AddDependency(mctx.Module(), generatedSourceTag, b.Generated_sources...)
		mctx.AddDependency(mctx.Module(), generatedHeaderTag, b.Generated_headers...)
		mctx.AddDependency(mctx.Module(), generatedDepTag, b.Generated_deps...)
	}

	// Things that a generated/transformed source depends on
	if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		if gsc.Properties.Host_bin != "" {
			parseAndAddVariationDeps(mctx, hostToolBinTag,
				gsc.Properties.Host_bin)
		}
		// Generated sources can use the outputs of another generated
		// source or library as a source file or dependency.
		parseAndAddVariationDeps(mctx, generatedDepTag,
			gsc.Properties.Module_deps...)
		parseAndAddVariationDeps(mctx, generatedSourceTag,
			gsc.Properties.Module_srcs...)
		parseAndAddVariationDeps(mctx, encapsulatesTag,
			gsc.Properties.Encapsulates...)
	}
}

func encapsulatesMutator(mctx blueprint.TopDownMutatorContext) {
	mainModule := mctx.Module()
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			return // Not enabled, so don't add dependencies
		}
	}

	mainGenProp, ok := getGenerateCommon(mainModule)
	if !ok {
		return
	}

	mctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		if mctx.OtherModuleDependencyTag(child) != encapsulatesTag {
			return false
		}
		childProp, ok := getGenerateCommon(child)
		if !ok {
			panic(errors.New(child.Name() + " does not support being encapsulated"))
		}

		mainGenProp.Properties.Export_gen_include_dirs = utils.AppendUnique(mainGenProp.Properties.Export_gen_include_dirs,
			childProp.Properties.Export_gen_include_dirs)
		return true
	})
}
