/*
 * Copyright 2018-2023 Arm Limited.
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
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
)

var (
	generatedHeaderTag       = dependencyTag{name: "generated_headers"}
	exportGeneratedHeaderTag = dependencyTag{name: "export_generated_headers"}
	generatedSourceTag       = dependencyTag{name: "generated_sources"}
	generatedDepTag          = dependencyTag{name: "generated_dep"}
	hostToolBinTag           = dependencyTag{name: "host_tool_bin"}
	filegroupTag             = dependencyTag{name: "filegroup"}
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

// GenerateProps contains the module properties that allow generation of
// output from arbitrary commands
type GenerateProps struct {
	LegacySourceProps
	AliasableProps
	EnableableProps
	InstallableProps

	/* The command that is to be run for this source generation.
	 * Substitutions can be made in the command, by using $name_of_var. A list of substitutions that can be used:
	 * $gen_dir      - the path to the directory which belongs to this source generator
	 * $in           - the path to the sources - space-delimited
	 * $out          - the path to the targets - space-delimited
	 * $depfile      - the path to generated dependency file
	 * $args         - the value of "args" - space-delimited
	 * $tool         - the path to the tool
	 * $tool <label> - the path to the tool with name <label>
	 * $host_bin     - the path to the binary that is produced by the host_bin module
	 * $(dep)_out    - the outputs of the generated_dep `dep`
	 * $src_dir      - the path to the project source directory - this will be different than the build source directory
	 *                 for Android.
	 * $module_dir   - the path to the module directory */
	Cmd *string

	// A paths to the tool that are to be used in cmd. If $tool is in the command variable, then this will be replaced
	// with the path to this tool. ${tool} refers to the first tool in a list. To reference
	// other tool use index syntax ${tool <label>} (e.g. ${tool fixer.py} for `fixer.py` tool from list).
	Tools []string

	// Adds a dependency on a binary with `host_supported: true` which is used by this module.
	// The path can be referenced in cmd as ${host_bin}.
	Host_bin *string

	// Values to use on Android for LOCAL_MODULE_TAGS, defining which builds this module is built for
	// TODO: Hide this in Android-specific properties
	Tags []string

	// A list of other modules that this generator depends on. The dependencies can be used in the command through
	// $name_of_dependency_dir .
	Generated_deps []string

	// A list of other modules that this generator depends on. The dependencies will be add to the list of srcs
	Generated_sources []string

	// A list of args that will be spaceseparated and add to the cmd
	Args []string

	// Used to indicate that the console should be used.
	Console *bool

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
	Target TgtType

	// If true, depfile name will be generated and can be used as ${depfile} reference in 'cmd'
	Depfile *bool

	// If set, Ninja will expand the string and write it to a file just
	// before executing the command. This can be used to e.g. contain ${in},
	// in cases where the command line length is a limiting factor.
	Rsp_content *string
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

/*
	We are swapping from bob_generate_source to bob_genrule

bob_genrule is made to be a stricter version that is compatible with Android.
For easiest compatibility, we are using Androids format for genrule.
Some properties in the struct may not be useful, but it is better to expose as many
features as possible rather than too few. Some are commented out as they would take special
implementation for features we do not already have in place.
*/
type AndroidGenerateRuleProps struct {
	Out []string
}

type AndroidGenerateCommonProps struct {
	// See https://ci.android.com/builds/submitted/8928481/linux/latest/view/soong_build.html
	Name                string
	Srcs                []string
	Exclude_srcs        []string
	Cmd                 *string
	Depfile             *bool
	Enabled             *bool
	Export_include_dirs []string
	Tool_files          []string
	Tools               []string
}

type androidGenerateCommon struct {
	moduleBase
	EnableableProps
	simpleOutputProducer
	headerProducer
	Properties struct {
		AndroidGenerateCommonProps
	}
}

type androidGenerateRule struct {
	androidGenerateCommon
	Properties struct {
		AndroidGenerateRuleProps
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

func (m *androidGenerateRule) shortName() string {
	return m.Name()
}

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

func (m *androidGenerateCommon) getAndroidGenerateCommon() *androidGenerateCommon {
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

// Return a list of headers generated by this module with full paths
func getHeadersGenerated(m dependentInterface) []string {
	return append(m.outputs(), m.implicitOutputs()...)
}

// Return a list of source files (not headers) generated by this module with full paths
func getSourcesGenerated(m dependentInterface) []string {
	return append(m.outputs(), m.implicitOutputs()...)
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getGenerateCommonInterface interface {
	getGenerateCommon() *generateCommon
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getAndroidGenerateCommonInterface interface {
	getAndroidGenerateCommon() *androidGenerateCommon
}

func getGenerateCommon(i interface{}) (*generateCommon, bool) {
	var gsc *generateCommon
	gsd, ok := i.(getGenerateCommonInterface)
	if ok {
		gsc = gsd.getGenerateCommon()
	}
	return gsc, ok
}

func getAndroidGenerateCommon(i interface{}) (*androidGenerateCommon, bool) {
	var gsc *androidGenerateCommon
	gsd, ok := i.(getAndroidGenerateCommonInterface)
	if ok {
		gsc = gsd.getAndroidGenerateCommon()
	}
	return gsc, ok
}

func (m *generateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *androidGenerateRule) getEnableableProps() *EnableableProps {
	return &m.EnableableProps
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

// GenerateSourceProps are properties of 'bob_generate_source', i.e. a module
// type which can generate sources using a single execution
// The command will be run once - with $in being the paths in "srcs" and $out being the paths in "out".
// The working directory will be the source directory, and all paths will be relative to the source directory
// if not else noted.
type GenerateSourceProps struct {
	// The list of files that will be output.
	Out []string
	// List of implicit sources. Implicit sources are input files that do not get
	// mentioned on the command line, and are not specified in the explicit sources.
	Implicit_srcs []string
	// Implicit source files that should not be included. Use with care.
	Exclude_implicit_srcs []string
}

func (g *GenerateSourceProps) getImplicitSources(ctx blueprint.BaseModuleContext) []string {
	return glob(ctx, g.Implicit_srcs, g.Exclude_implicit_srcs)
}

type generateSource struct {
	generateCommon
	Properties struct {
		GenerateSourceProps
	}
}

// generateSource supports installation
var _ installable = (*generateSource)(nil)

func (m *generateSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.generateSourceActions(m, ctx)
	}
}

func (m *generateSource) FeaturableProperties() []interface{} {
	return append(m.generateCommon.FeaturableProperties(), &m.Properties.GenerateSourceProps)
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

// Returns the outputs of the generated dependencies of a module. This is used for more complex
// dependencies, where the dependencies are not just binaries or headers, but where the paths are
// used directly in a script
func getDependentArgsAndFiles(ctx blueprint.ModuleContext, args map[string]string) (depfiles []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(m) == generatedDepTag
		},
		func(m blueprint.Module) {
			gen, ok := m.(dependentInterface)
			if !ok {
				utils.Die("%s is not a valid dependent interface", reflect.TypeOf(m).String())
			}

			depName := ctx.OtherModuleName(m)
			// When the dependent module is another Bob generated
			// module, provide all its outputs so the using module can
			// pick and choose what it uses.
			if gc, ok := getGenerateCommon(m); ok {
				args[depName+"_out"] = strings.Join(gc.outputs(), " ")
			} else {
				args[depName+"_out"] = strings.Join(gen.outputs(), " ")
			}

			depfiles = append(depfiles, gen.outputs()...)
			depfiles = append(depfiles, gen.implicitOutputs()...)
		})

	return
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

func getDepfileName(s string) string {
	return utils.FlattenPath(s) + ".d"
}

func getRspfileName(s string) string {
	return "." + utils.FlattenPath(s) + ".rsp"
}

func (m *generateSource) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.Implicit_srcs = utils.PrefixDirs(m.Properties.Implicit_srcs, projectModuleDir(ctx))
	m.Properties.Exclude_implicit_srcs = utils.PrefixDirs(m.Properties.Exclude_implicit_srcs, projectModuleDir(ctx))
	m.generateCommon.processPaths(ctx, g)
}

// Return an inouts structure naming all the files associated with a
// generateSource's inputs.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies
// to out, depfile and rspfile. The output directory (if needed) needs to be
// added in by the backend specific GenerateBuildAction()
func (m *generateSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var io inout
	io.in = append(getBackendPathsInSourceDir(g, m.getSourcesResolved(ctx)),
		getGeneratedFiles(ctx)...)
	io.out = m.Properties.Out
	io.implicitSrcs = getBackendPathsInSourceDir(g, m.Properties.getImplicitSources(ctx))

	if depfile, ok := m.getDepfile(); ok {
		io.depfile = depfile
	}
	if rspfile, ok := m.getRspfile(); ok {
		io.rspfile = rspfile
	}

	return []inout{io}
}

func (m *generateSource) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	// Install everything that we generate
	return m.outputs()
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
		// List of implicit sources. Implicit sources are input files that do not get mentioned on the command line,
		// and are not specified in the explicit sources.
		Implicit_srcs []string
	}
}

func (tsp *TransformSourceProps) inoutForSrc(re *regexp.Regexp, source filePath, depfile *bool, rspfile bool) (io inout) {
	io.in = []string{source.buildPath()}

	for _, rep := range tsp.Out.Replace {
		out := filepath.Join(re.ReplaceAllString(source.localPath(), rep))
		io.out = append(io.out, out)
	}

	if proptools.Bool(depfile) {
		io.depfile = getDepfileName(source.localPath())
	}

	for _, implSrc := range tsp.Out.Implicit_srcs {
		implSrc = re.ReplaceAllString(source.localPath(), implSrc)
		io.implicitSrcs = append(io.implicitSrcs, filepath.Join(source.moduleDir(), implSrc))
	}

	if rspfile {
		io.rspfile = getRspfileName(source.localPath())
	}

	return
}

func (m *transformSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.transformSourceActions(m, ctx)
	}
}

func (m *androidGenerateRule) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.androidGenerateRuleActions(m, ctx)
	}
}

func (m *transformSource) FeaturableProperties() []interface{} {
	return append(m.generateCommon.FeaturableProperties(), &m.Properties.TransformSourceProps)
}

func (m *transformSource) sourceInfo(ctx blueprint.ModuleContext, g generatorBackend) []filePath {
	var sourceList []filePath
	for _, src := range m.getSourcesResolved(ctx) {
		sourceList = append(sourceList, newSourceFilePath(src, ctx, g))
	}
	for _, src := range getGeneratedFiles(ctx) {
		sourceList = append(sourceList, newGeneratedFilePath(src))
	}
	return sourceList
}

// Return an inouts structure naming all the files associated with
// each transformSource input.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies
// to out, depfile and rspfile. The output directory (if needed) needs to be
// added in by the backend specific GenerateBuildAction()
func (m *transformSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var inouts []inout
	re := regexp.MustCompile(m.Properties.Out.Match)

	for _, source := range m.sourceInfo(ctx, g) {
		io := m.Properties.inoutForSrc(re, source, m.generateCommon.Properties.Depfile,
			m.generateCommon.Properties.Rsp_content != nil)
		inouts = append(inouts, io)
	}

	return inouts
}

func (m *transformSource) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	// Install everything that we generate
	return m.outputs()
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
}

// transformSource supports installation
var _ installable = (*transformSource)(nil)

func generateSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateSource{}
	module.generateCommon.init(&config.Properties,
		GenerateProps{}, GenerateSourceProps{})

	return module, []interface{}{&module.generateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}

func transformSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &transformSource{}
	module.generateCommon.init(&config.Properties,
		GenerateProps{}, TransformSourceProps{})

	return module, []interface{}{&module.generateCommon.Properties,
		&module.Properties,
		&module.SimpleName.Properties}
}

func generateRuleAndroidFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &androidGenerateRule{}

	return module, []interface{}{&module.androidGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}

// ModuleContext Helpers

// Return the outputs() of all GeneratedSource dependencies of the current
// module. The current module can be generated or a library, and the
// dependencies can be anything implementing DependentInterface (so "generated"
// is a misnomer, because this includes libraries, too).
func getGeneratedFiles(ctx blueprint.ModuleContext) []string {
	var srcs []string
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == generatedSourceTag },
		func(m blueprint.Module) {
			if gs, ok := m.(dependentInterface); ok {
				srcs = append(srcs, gs.outputs()...)
				srcs = append(srcs, gs.implicitOutputs()...)
			} else {
				utils.Die("%s does not have outputs", ctx.OtherModuleName(m))
			}
		})
	return srcs
}

func generatedDependerMutator(mctx blueprint.BottomUpMutatorContext) {

	if _, ok := mctx.Module().(*generateSource); ok {
		getBackend(mctx).getLogger().Warn(warnings.GenerateRuleWarning, mctx.BlueprintsFile(), mctx.ModuleName())
	}

	if e, ok := mctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	// Things which depend on generated/transformed sources
	if l, ok := getLibrary(mctx.Module()); ok {
		mctx.AddDependency(mctx.Module(), generatedSourceTag, l.Properties.Generated_sources...)
		mctx.AddDependency(mctx.Module(), generatedHeaderTag, l.Properties.Generated_headers...)
		mctx.AddDependency(mctx.Module(), exportGeneratedHeaderTag, l.Properties.Export_generated_headers...)
		mctx.AddDependency(mctx.Module(), generatedDepTag, l.Properties.Generated_deps...)
	}

	// Things that a generated/transformed source depends on
	if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		if gsc.Properties.Host_bin != nil {
			parseAndAddVariationDeps(mctx, hostToolBinTag,
				proptools.String(gsc.Properties.Host_bin))
		}
		// Generated sources can use the outputs of another generated
		// source or library as a source file or dependency.
		parseAndAddVariationDeps(mctx, generatedDepTag,
			gsc.Properties.Generated_deps...)
		parseAndAddVariationDeps(mctx, generatedSourceTag,
			gsc.Properties.Generated_sources...)
	}

	// New Android Genrules can depend on source of other modules with a different format.
	// We must transform the format if we are not generating for Android.
	if _, ok := getBackend(mctx).(*linuxGenerator); ok {
		if agsc, ok := getAndroidGenerateCommon(mctx.Module()); ok {
			var removeList []string
			for _, s := range agsc.Properties.Srcs {
				if s[0] == ':' {
					parseAndAddVariationDeps(mctx, generatedSourceTag,
						s[1:])
					parseAndAddVariationDeps(mctx, generatedDepTag,
						s[1:])
					removeList = append(removeList, s)
				}
			}
			for i := range removeList {
				agsc.Properties.Srcs = append(agsc.Properties.Srcs[:i], agsc.Properties.Srcs[i+1:]...)
			}
		}
	}

	// These rules also need to support variants when depending on tools. This strictly breaks android's genrule definition.
	// However, if a colon appears at the end of a module name with a text string, we assume there is a variant
	// called <module_name>__<variant_name> generated. Which bob currently does. This will fix behaviour on Android, to
	// ensure it works on Linux, the backend must see this as a generated_dep which is processing done in the linux backend.
	if agsc, ok := getAndroidGenerateCommon((mctx.Module())); ok {
		var removeList []string
		for _, s := range agsc.Properties.Tools {
			if strings.Contains(s, ":") {
				if _, ok := getBackend(mctx).(*linuxGenerator); ok {
					parseAndAddVariationDeps(mctx, generatedDepTag,
						s)
				} else {
					agsc.Properties.Tools = append(agsc.Properties.Tools, strings.Replace(s, ":", "__", 1))
					removeList = append(removeList, s)
				}
			}
		}
		for i := range removeList {
			agsc.Properties.Tools = append(agsc.Properties.Tools[:i], agsc.Properties.Tools[i+1:]...)
		}
	}

	// Convert any filegroup dependencies into the correct format
	if _, ok := getBackend(mctx).(*androidBpGenerator); ok {
		if l, ok := getLibrary(mctx.Module()); ok {
			l.Properties.Srcs = append(l.Properties.Srcs, utils.PrefixAll(l.Properties.Filegroup_srcs, ":")...)
		}
	}

}
