//go:build soong
// +build soong

/*
 * Copyright 2020-2023 Arm Limited.
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
package genrulebob

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/genrule"

	"github.com/ARM-software/bob-build/internal/soong_compat"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
)

type commonProps struct {
	Srcs                    []string
	Export_gen_include_dirs []string
	Cmd                     string
	Host_bin                string
	Tools                   []string
	Depfile                 bool
	Generated_deps          []string
	Generated_sources       []string
	Encapsulates            []string
	Cflags                  []string
	Conlyflags              []string
	Cxxflags                []string
	Asflags                 []string
	Ldflags                 []string
	Ldlibs                  []string
	Rsp_content             *string

	// if install path is not empty, module will be installed onto partition,
	// it should contain path relative to partition root
	Install_path *string
	// Partition selection options
	Install_in_data      bool
	Install_in_testcases bool
}

type genruleProps struct {
	Out           []string
	Implicit_srcs []string
	Implicit_outs []string
}

type gensrcsProps struct {
	Out struct {
		Match         string
		Replace       []string
		Implicit_srcs []string
		Implicit_outs []string
	}
}

// inout structure with soong data types
type soongInout struct {
	in           android.Paths
	out          android.WritablePaths
	depfile      android.WritablePath
	implicitSrcs android.Paths
	implicitOuts android.WritablePaths
	rspfile      android.WritablePath
}

// helper interface to distinguish genrulebob/gensrcsbob module from other soong modules
type genruleInterface interface {
	genrule.SourceFileGenerator

	outputs() android.WritablePaths
	implicitOutputs() android.WritablePaths
	outputPath() android.Path
}

type genrulebobCommon struct {
	android.ModuleBase

	Properties commonProps

	genDir               android.Path
	exportGenIncludeDirs android.Paths
	inouts               []soongInout
	installedOuts        []android.InstallPath
}

type genrulebob struct {
	genrulebobCommon
	Properties genruleProps
}

type gensrcsbob struct {
	genrulebobCommon
	Properties gensrcsProps
}

// implemented interfaces check
var _ android.AndroidMkEntriesProvider = (*genrulebobCommon)(nil)
var _ genruleInterface = (*genrulebobCommon)(nil)
var _ android.Module = (*genrulebob)(nil)
var _ android.Module = (*gensrcsbob)(nil)
var _ android.SourceFileProducer = (*genrulebob)(nil)

type generatedSourceTagType struct {
	blueprint.BaseDependencyTag
}

type generatedDepTagType struct {
	blueprint.BaseDependencyTag
}

type hostToolBinTagType struct {
	blueprint.BaseDependencyTag
}

func init() {
	// Import config package into pctx context, which is used for writing ninja rules.
	// This makes vars from config package accessible, eg. ${config.ClangBin} reference
	// can be used in cmd and args of ninja rules, which will interpolate
	// into ${g.android.soong.cc.config.ClangBin}
	pctx.Import("android/soong/cc/config")
}

var (
	pctx = android.NewPackageContext("plugins/genrulebob")

	generatedSourceTag generatedSourceTagType
	generatedDepTag    generatedDepTagType
	hostToolBinTag     hostToolBinTagType
)

func genrulebobFactory() android.Module {
	m := &genrulebob{}
	// register all structs that contain module properties (parsable from .bp file)
	// note: we register our custom properties first, to take precedence before common ones
	m.AddProperties(&m.Properties)
	m.AddProperties(&m.genrulebobCommon.Properties)
	// init module with target-specific variants info, needed also to get install path right
	// (hardcode to device variant, host variant currently not supported)
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)
	return m
}

func gensrcsbobFactory() android.Module {
	m := &gensrcsbob{}
	// register all structs that contain module properties (parsable from .bp file)
	// note: we register our custom properties first, to take precedence before common ones
	m.AddProperties(&m.Properties)
	m.AddProperties(&m.genrulebobCommon.Properties)
	// init module with target-specific variants info, needed also to get install path right
	// (hardcode to device variant, host variant currently not supported)
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)
	return m
}

func init() {
	android.RegisterModuleType("genrule_bob", genrulebobFactory)
	android.RegisterModuleType("gensrcs_bob", gensrcsbobFactory)
}

func (m *genrulebobCommon) outputPath() android.Path {
	return m.genDir
}

func (m *genrulebobCommon) outputs() (ret android.WritablePaths) {
	for _, io := range m.inouts {
		ret = append(ret, io.out...)
	}
	return
}

func (m *genrulebobCommon) implicitOutputs() (ret android.WritablePaths) {
	for _, io := range m.inouts {
		ret = append(ret, io.implicitOuts...)
	}
	return
}

func (m *genrulebobCommon) allOutputs() (ret android.WritablePaths) {
	for _, io := range m.inouts {
		ret = append(ret, io.out...)
		ret = append(ret, io.implicitOuts...)
	}
	return
}

// Soong's gen dirs are generally of the form `/path/to/module/gen`. However, the
// Linux backend uses the form `build/gen/module_name`. Normally this doesn't matter,
// as everything is contained within the gen dir, except when chaining multiple
// generated modules. In this case, bob_transform_source used on Linux may expect the
// module name to be included when doing the regex replacement, and be exporting
// include directories accordingly. We therefore need to add a subdirectory named after
// the module inside Soong's gen dir for compatibility.
func pathForModuleGen(ctx android.ModuleContext, paths ...string) android.WritablePath {
	prefix := []string{ctx.ModuleName()}
	return android.PathForModuleGen(ctx, append(prefix, paths...)...)
}

func pathsForModuleGen(ctx android.ModuleContext, paths []string) (ret android.WritablePaths) {
	for _, path := range paths {
		ret = append(ret, pathForModuleGen(ctx, path))
	}
	return
}

// GeneratedSourceFiles, GeneratedHeaderDirs and GeneratedDeps implement the
// genrule.SourceFileGenerator interface, which allows these modules to be used
// to generate inputs for cc_library and cc_binary modules.
func (m *genrulebobCommon) GeneratedSourceFiles() android.Paths {
	return m.allOutputs().Paths()
}

func (m *genrulebobCommon) GeneratedHeaderDirs() android.Paths {
	return m.exportGenIncludeDirs
}

func (m *genrulebobCommon) GeneratedDeps() (srcs android.Paths) {
	return m.allOutputs().Paths()
}

// Srcs implements the android.SourceFileProducer interface, which allows
// the outputs of these modules to be referenced using the `:module` syntax.
func (m *genrulebobCommon) Srcs() android.Paths {
	return m.outputs().Paths()
}

func (m *genrulebobCommon) DepsMutator(ctx android.BottomUpMutatorContext) {
	if m.Properties.Host_bin != "" {
		ctx.AddFarVariationDependencies(ctx.Config().BuildOSTarget.Variations(),
			hostToolBinTag, m.Properties.Host_bin)
	}

	// `generated_deps` and `generated_sources` can refer not only to source
	// generation modules, but to binaries and libraries. In this case we
	// need to handle multilib builds, where a 'target' library could be
	// split into 32 and 64-bit variants. Use `AddFarVariationDependencies`
	// here, because this will automatically choose the first available
	// variant, rather than the other dependency-adding functions, which
	// will error when multiple variants are present.
	ctx.AddFarVariationDependencies(nil, generatedDepTag, m.Properties.Generated_deps...)
	ctx.AddFarVariationDependencies(nil, generatedSourceTag, m.Properties.Generated_sources...)
}

func (m *genrulebobCommon) getHostBin(ctx android.ModuleContext) android.OptionalPath {
	if m.Properties.Host_bin == "" {
		return android.OptionalPath{}
	}
	hostBinModule := ctx.GetDirectDepWithTag(m.Properties.Host_bin, hostToolBinTag)
	htp, ok := hostBinModule.(genrule.HostToolProvider)
	if !ok {
		panic(fmt.Errorf("%s is not a host tool", m.Properties.Host_bin))
	}
	return htp.HostToolPath()
}

func (m *genrulebobCommon) getArgs(ctx android.ModuleContext) (args map[string]string, dependents []android.Path) {
	args = map[string]string{
		"gen_dir":         pathForModuleGen(ctx).String(),
		"asflags":         utils.Join(m.Properties.Asflags),
		"cflags":          utils.Join(m.Properties.Cflags),
		"conlyflags":      utils.Join(m.Properties.Conlyflags),
		"cxxflags":        utils.Join(m.Properties.Cxxflags),
		"ldflags":         utils.Join(m.Properties.Ldflags),
		"ldlibs":          utils.Join(m.Properties.Ldlibs),
		"module_dir":      ctx.ModuleDir(),
		"shared_libs_dir": "",

		// flag_defaults is primarily used to invoke sub-makes of
		// different libraries. This shouldn't be needed on Android.
		// This means the following can't be expanded:
		"ar":     "",
		"as":     "",
		"cc":     "",
		"cxx":    "",
		"linker": "",
	}

	// Add arguments providing information about other modules the current
	// one depends on, accessible via ${module}_out.
	ctx.VisitDirectDepsWithTag(generatedDepTag, func(dep android.Module) {
		// If a generated module depends on a library/binary which was split
		// into host/target variants by the Android.bp generator, they will
		// have a target-specific suffix; strip this so that variable
		// references in `cmd` are still correct.
		varName := strings.TrimSuffix(strings.TrimSuffix(dep.Name(), "__host"), "__target")

		if gdep, ok := dep.(genruleInterface); ok {
			dependents = append(dependents, gdep.outputs().Paths()...)
			dependents = append(dependents, gdep.implicitOutputs().Paths()...)
			args[varName+"_out"] = utils.Join(gdep.outputs().Strings())

		} else if ccmod, ok := dep.(cc.LinkableInterface); ok {
			out := ccmod.OutputFile()
			dependents = append(dependents, out.Path())
			args[varName+"_out"] = out.String()
		}
	})

	return
}

func (m *genrulebobCommon) getModuleSrcs(ctx android.ModuleContext) (srcs []android.Path) {
	ctx.VisitDirectDepsWithTag(generatedSourceTag, func(dep android.Module) {
		if gdep, ok := dep.(genruleInterface); ok {
			srcs = append(srcs, gdep.outputs().Paths()...)
			srcs = append(srcs, gdep.implicitOutputs().Paths()...)
		} else if ccmod, ok := dep.(cc.LinkableInterface); ok {
			srcs = append(srcs, ccmod.OutputFile().Path())
		}
	})
	return
}

var touchRule = pctx.StaticRule("touch",
	blueprint.RuleParams{
		Command:     "touch -c $out",
		Description: "touch $out",
	})

func (m *genrulebobCommon) writeNinjaRules(ctx android.ModuleContext, args map[string]string) {
	ruleparams := blueprint.RuleParams{
		Command: m.Properties.Cmd,
		Restat:  true,
	}

	if m.Properties.Rsp_content != nil {
		args["rspfile"] = ""
		ruleparams.Rspfile = "${rspfile}"
		ruleparams.RspfileContent = *m.Properties.Rsp_content
	}

	// no need to keep depfile in args, as the same named argument will be provided by ninja BuildParams below,
	// we need it however to check existence of cmd argument
	keys := utils.SortedKeys(args)
	if m.Properties.Depfile {
		keys = append(keys, "depfile")
		ruleparams.Deps = blueprint.DepsGCC
	}

	rule := ctx.Rule(pctx, "bob_gen_"+ctx.ModuleName(), ruleparams, keys...)

	for _, io := range m.inouts {
		// `args` is slightly different for each inout, but blueprint's
		// parseBuildParams() function makes a deep copy of the map, so
		// we're OK to re-use it for each target.
		if m.Properties.Rsp_content != nil {
			args["rspfile"] = io.rspfile.String()
		}

		mainRuleOuts := io.out
		mainRuleImplicitOuts := io.implicitOuts

		// ninja currently does not support the case when depfile is
		// defined and multiple outputs at the same time. So adjust the
		// main rule to have a single output, and link the remaining
		// outputs using a separate rule.
		if m.Properties.Depfile && (len(io.out)+len(io.implicitOuts)) > 1 {
			// No-op rule linking the extra outputs to the main
			// output. Update the extra outputs' mtime in case the
			// script actually creates the extra outputs first.

			allOutputs := append(mainRuleOuts, mainRuleImplicitOuts...)
			mainRuleOuts = allOutputs[0:1]
			mainRuleImplicitOuts = []android.WritablePath{}

			ctx.Build(pctx,
				android.BuildParams{
					Rule:    touchRule,
					Inputs:  mainRuleOuts.Paths(),
					Outputs: allOutputs[1:],
				})
		}

		ctx.Build(pctx,
			android.BuildParams{
				Rule:            rule,
				Description:     "gen " + ctx.ModuleName(),
				Inputs:          io.in,
				Implicits:       io.implicitSrcs,
				Outputs:         mainRuleOuts,
				ImplicitOutputs: mainRuleImplicitOuts,
				Args:            args,
				Depfile:         io.depfile,
			})

		if m.Properties.Install_path != nil {
			for _, outfile := range io.out {
				// generate ninja rule for copying file onto partition
				m.installedOuts = append(m.installedOuts, ctx.InstallFile(android.PathForModuleInstall(ctx, *m.Properties.Install_path), filepath.Base(outfile.String()), outfile))
			}
		}
	}
}

func (m *genrulebobCommon) calcExportGenIncludeDirs(ctx android.ModuleContext) android.Paths {
	var allIncludeDirs android.Paths

	// Add our own include dirs
	for _, dir := range m.Properties.Export_gen_include_dirs {
		allIncludeDirs = append(allIncludeDirs, pathForModuleGen(ctx, dir))
	}

	// Add include dirs of our all dependencies
	ctx.WalkDeps(func(child android.Module, parent android.Module) bool {
		if cmod, ok := child.(genruleInterface); ok {
			for _, dir := range cmod.GeneratedHeaderDirs() {
				allIncludeDirs = append(allIncludeDirs, dir)
			}
		}
		return true
	})

	// Make unique items as for recursive passes it may contain redundant ones
	return android.FirstUniquePaths(allIncludeDirs)
}

func getDepfileName(s string) string {
	return utils.FlattenPath(s) + ".d"
}

func getRspfileName(s string) string {
	return "." + utils.FlattenPath(s) + ".rsp"
}

// Remove the relative part from android.Path
func nonRelPathString(path android.Path) string {
	return strings.TrimSuffix(path.String(), path.Rel())
}

func pathsForImplicitSrcs(ctx android.ModuleContext, source android.Path, props []string) (paths android.Paths) {
	if mgp, ok := source.(android.ModuleGenPath); ok {
		for _, prop := range props {
			paths = append(paths, mgp.InSameDir(ctx, prop))
		}
	} else {
		nonRel := nonRelPathString(source)
		for _, prop := range props {
			paths = append(paths, android.PathForSource(ctx, filepath.Join(nonRel, prop)))
		}
	}
	return
}

func (m *gensrcsbob) inoutForSrc(ctx android.ModuleContext, re *regexp.Regexp,
	source android.Path, commonImplicits android.Paths) (io soongInout) {

	// helper to replace source path
	replaceSource := func(props []string) (newProps []string) {
		for _, prop := range props {
			newProps = append(newProps, re.ReplaceAllString(source.Rel(), prop))
		}
		return
	}

	io.in = android.Paths{source}
	io.out = pathsForModuleGen(ctx, replaceSource(m.Properties.Out.Replace))
	io.implicitSrcs = append(pathsForImplicitSrcs(ctx, source, replaceSource(m.Properties.Out.Implicit_srcs)),
		commonImplicits...)
	io.implicitOuts = pathsForModuleGen(ctx, replaceSource(m.Properties.Out.Implicit_outs))

	if m.genrulebobCommon.Properties.Depfile {
		io.depfile = pathForModuleGen(ctx, getDepfileName(source.Rel()))
	}
	if m.genrulebobCommon.Properties.Rsp_content != nil {
		io.rspfile = pathForModuleGen(ctx, getRspfileName(source.Rel()))
	}

	return
}

func (m *gensrcsbob) createInouts(ctx android.ModuleContext,
	commonImplicits android.Paths) (inouts []soongInout) {
	re := regexp.MustCompile(m.Properties.Out.Match)

	for _, src := range m.genrulebobCommon.Properties.Srcs {
		inouts = append(inouts,
			m.inoutForSrc(ctx, re, android.PathForModuleSrc(ctx, src), commonImplicits))
	}
	for _, src := range m.getModuleSrcs(ctx) {
		inouts = append(inouts,
			m.inoutForSrc(ctx, re, src, commonImplicits))
	}

	return
}

func (m *genrulebob) createInouts(ctx android.ModuleContext,
	commonImplicits android.Paths) []soongInout {

	io := soongInout{
		in:           append(android.PathsForModuleSrc(ctx, m.genrulebobCommon.Properties.Srcs), m.getModuleSrcs(ctx)...),
		implicitSrcs: append(commonImplicits, android.PathsForModuleSrc(ctx, m.Properties.Implicit_srcs)...),
		out:          pathsForModuleGen(ctx, m.Properties.Out),
		implicitOuts: pathsForModuleGen(ctx, m.Properties.Implicit_outs),
	}
	if m.genrulebobCommon.Properties.Depfile {
		io.depfile = pathForModuleGen(ctx, getDepfileName(m.Name()))
	}
	if m.genrulebobCommon.Properties.Rsp_content != nil {
		io.rspfile = pathForModuleGen(ctx, getRspfileName(m.Name()))
	}

	return []soongInout{io}
}

func (m *genrulebobCommon) setupBuildActions(ctx android.ModuleContext) (args map[string]string, implicits []android.Path) {
	args, implicits = m.getArgs(ctx)

	m.genDir = pathForModuleGen(ctx)
	m.exportGenIncludeDirs = m.calcExportGenIncludeDirs(ctx)

	if hostBin := m.getHostBin(ctx); hostBin.Valid() {
		args["host_bin"] = hostBin.String()
		implicits = append(implicits, hostBin.Path())
	}

	cmd, toolArgs, dependentTools := m.processCmdTools(ctx, m.Properties.Cmd)

	m.Properties.Cmd = cmd
	for k, v := range toolArgs {
		args[k] = v
	}

	implicits = append(implicits, dependentTools...)

	return
}

func (m *genrulebobCommon) processCmdTools(ctx android.ModuleContext, cmd string) (string, map[string]string, []android.Path) {

	dependentTools := []android.Path{}
	toolsLabels := map[string]android.Path{}
	args := map[string]string{}
	firstTool := ""

	addToolsLabel := func(label string, tool android.Path) {
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
			toolPath := android.PathForModuleSrc(ctx, tool)
			addToolsLabel(tool, toolPath)
			dependentTools = append(dependentTools, toolPath)
		}
	}

	// add first tool for ${tool}
	if utils.ContainsArg(cmd, "tool") {
		args["tool"] = toolsLabels[firstTool].String()
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
			args[toolKey] = toolPath.String()
			idx++
		} else {
			ctx.ModuleErrorf("unknown tool %q in tools.", submatch[1])
		}
	}

	return cmd, args, dependentTools
}

func (m *genrulebob) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	args, implicits := m.setupBuildActions(ctx)
	m.inouts = m.createInouts(ctx, implicits)
	m.writeNinjaRules(ctx, args)
}

func (m *gensrcsbob) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	args, implicits := m.setupBuildActions(ctx)
	m.inouts = m.createInouts(ctx, implicits)
	m.writeNinjaRules(ctx, args)
}

func (m *genrulebobCommon) AndroidMkEntries() []android.AndroidMkEntries {
	entries := []android.AndroidMkEntries{}
	outs := []android.OptionalPath{}
	reqs := []string{}

	// reference installed files instead of built files will ensure triggering install rule after build rule
	// but only for A12 and below. A13 Soong handles the install rules.
	if m.Properties.Install_path != nil && !soong_compat.SoongSupportsMkInstallTargets() {
		for _, outfile := range m.installedOuts {
			outs = append(outs, android.OptionalPathForPath(outfile))
		}
	} else {
		for _, io := range m.inouts {
			for _, outfile := range io.out {
				outs = append(outs, android.OptionalPathForPath(outfile))
			}
		}
	}

	for _, outfile := range outs {
		subname := ""

		// if module has more than one output, keep LOCAL_MODULE unique
		if len(outs) > 1 {
			subname = "__" + utils.FlattenPath(outfile.Path().Rel())
			reqs = append(reqs, m.Name()+subname)
		}

		entries = append(entries, android.AndroidMkEntries{
			Class:      "DATA",
			OutputFile: outfile,
			// if module has more than one output, keep LOCAL_MODULE unique
			SubName: subname,
			Include: "$(BUILD_PREBUILT)",
			ExtraEntries: soong_compat.ConvertAndroidMkExtraEntriesFunc(
				func(entries *android.AndroidMkEntries) {
					// don't install in data partition (which is enforced behavior when class is DATA)
					entries.SetBool("LOCAL_UNINSTALLABLE_MODULE", true)
				},
			),
		})
	}

	// If we have more than one output, we need a final target which groups all of the others.
	// Mimic what the native genrule generates using the ETC class.
	if len(outs) > 1 {
		entries = append(entries, android.AndroidMkEntries{
			SubName:    "",
			OutputFile: outs[0], // The entry needs an output file, otherwise it won't be emitted.
			Include:    "$(BUILD_PHONY_PACKAGE)",
			Required:   reqs,
		})
	}

	return entries
}

// required to generate ninja rule for copying file onto partition
func (m *genrulebobCommon) InstallBypassMake() bool {
	return m.Properties.Install_path != nil
}

func (m *genrulebobCommon) InstallInData() bool {
	return m.Properties.Install_in_data
}

func (m *genrulebobCommon) InstallInTestcases() bool {
	return m.Properties.Install_in_testcases
}
