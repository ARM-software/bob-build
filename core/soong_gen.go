// +build soong

/*
 * Copyright 2019 Arm Limited.
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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"android/soong/android"
	"android/soong/cc"
	"android/soong/genrule"

	"github.com/ARM-software/bob-build/utils"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type genBackendProps struct {
	Srcs                    []string
	Out                     []string
	Implicit_srcs           []string
	Implicit_outs           []string
	Export_gen_include_dirs []string
	Cmd                     string
	HostBin                 string
	Tool                    string
	Depfile                 bool
	Module_deps             []string
	Module_srcs             []string
	Encapsulates            []string
	Cflags                  []string
	Conlyflags              []string
	Cxxflags                []string
	Asflags                 []string
	Ldflags                 []string
	Ldlibs                  []string

	Transform_srcs []string
	TransformSourceProps
}

type genBackend struct {
	android.ModuleBase
	Properties genBackendProps

	genDir               android.Path
	exportGenIncludeDirs android.WritablePaths
	inouts               []soongInout
}

func genBackendFactory() android.Module {
	m := &genBackend{}

	m.AddProperties(&m.Properties)
	android.InitAndroidModule(m)

	return m
}

func (m *genBackend) outputs() (ret android.WritablePaths) {
	for _, io := range m.inouts {
		ret = append(ret, io.out...)
		ret = append(ret, io.implicitOuts...)
	}
	return
}

func (m *genBackend) filterOutputs(predicate func(string) bool) (ret android.Paths) {
	for _, p := range m.outputs() {
		if predicate(p.String()) {
			ret = append(ret, p)
		}
	}
	return
}

func pathsForModuleGen(ctx android.ModuleContext, paths []string) (ret android.WritablePaths) {
	for _, path := range paths {
		ret = append(ret, android.PathForModuleGen(ctx, path))
	}
	return
}

// GeneratedSourceFiles, GeneratedHeaderDirs and GeneratedDeps implement the
// genrule.SourceFileGenerator interface, which allows these modules to be used
// to generate inputs for cc_library and cc_binary modules.
func (m *genBackend) GeneratedSourceFiles() android.Paths {
	return m.filterOutputs(utils.IsCompilableSource)
}

func (m *genBackend) GeneratedHeaderDirs() android.Paths {
	return m.exportGenIncludeDirs.Paths()
}

func (m *genBackend) GeneratedDeps() (srcs android.Paths) {
	return m.filterOutputs(utils.IsNotCompilableSource)
}

func (m *genBackend) DepsMutator(mctx android.BottomUpMutatorContext) {
	if m.Properties.HostBin != "" {
		mctx.AddFarVariationDependencies([]blueprint.Variation{
			{Mutator: "arch", Variation: mctx.Config().BuildOsVariant}},
			hostToolBinTag, m.Properties.HostBin)
	}

	parseAndAddVariationDeps(mctx, generatedDepTag,
		m.Properties.Module_deps...)
	parseAndAddVariationDeps(mctx, generatedSourceTag,
		m.Properties.Module_srcs...)
	parseAndAddVariationDeps(mctx, encapsulatesTag,
		m.Properties.Encapsulates...)
}

func (m *genBackend) getHostBin(ctx android.ModuleContext) string {
	if m.Properties.HostBin == "" {
		return ""
	}
	hostBinModule := ctx.GetDirectDepWithTag(m.Properties.HostBin, hostToolBinTag)
	htp, ok := hostBinModule.(genrule.HostToolProvider)
	if !ok {
		panic(fmt.Errorf("%s is not a host tool", m.Properties.HostBin))
	}
	return htp.HostToolPath().String()
}

func (m *genBackend) getArgs(ctx android.ModuleContext) (args map[string]string, dependents []android.Path) {
	dependents = android.PathsForSource(ctx, m.Properties.Implicit_srcs)
	args = map[string]string{
		"bob_config":      filepath.Join(getBuildDir(), configName),
		"bob_config_opts": configOpts,
		"gen_dir":         android.PathForModuleGen(ctx).String(),
		"host_bin":        m.getHostBin(ctx),
		"asflags":         utils.Join(m.Properties.Asflags),
		"cflags":          utils.Join(m.Properties.Cflags),
		"conlyflags":      utils.Join(m.Properties.Conlyflags),
		"cxxflags":        utils.Join(m.Properties.Cxxflags),
		"ldflags":         utils.Join(m.Properties.Ldflags),
		"ldlibs":          utils.Join(m.Properties.Ldlibs),

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
	// one depends on, accessible via ${module}_out and ${module}_dir.
	ctx.VisitDirectDepsWithTag(generatedDepTag, func(dep android.Module) {
		if gdep, ok := dep.(*genBackend); ok {
			outs := gdep.outputs()
			dependents = append(dependents, outs.Paths()...)

			args[dep.Name()+"_dir"] = gdep.genDir.String()
			args[dep.Name()+"_out"] = strings.Join(outs.Strings(), " ")
		} else if ccmod, ok := dep.(*cc.Module); ok {
			out := ccmod.OutputFile()
			dependents = append(dependents, out.Path())
			// We only expect to use the output from static/shared libraries
			// and binaries, so `_dir' is not supported on these.
			args[dep.Name()+"_out"] = out.String()
		}
	})

	return
}

type soongInout struct {
	in           android.Paths
	out          android.WritablePaths
	depfile      android.WritablePath
	implicitSrcs android.Paths
	implicitOuts android.WritablePaths
}

func (m *genBackend) buildInouts(ctx android.ModuleContext, args map[string]string) {
	if m.Properties.Depfile {
		args["depfile"] = ""
	}
	args["headers_generated"] = ""
	args["srcs_generated"] = ""

	rule := ctx.Rule(apctx,
		"bob_gen_"+ctx.ModuleName(),
		blueprint.RuleParams{
			Command: m.Properties.Cmd,
			Restat:  true,
		},
		utils.SortedKeys(args)...,
	)

	for _, sio := range m.inouts {
		// `args` is slightly different for each inout, but blueprint's
		// parseBuildParams() function makes a deep copy of the map, so
		// we're OK to re-use it for each target.
		if m.Properties.Depfile {
			args["depfile"] = sio.depfile.String()
		}
		args["headers_generated"] = strings.Join(utils.Filter(utils.IsHeader, sio.out.Strings()), " ")
		args["srcs_generated"] = strings.Join(utils.Filter(utils.IsNotHeader, sio.out.Strings()), " ")

		ctx.Build(apctx,
			android.BuildParams{
				Rule:            rule,
				Description:     "gen " + ctx.ModuleName(),
				Inputs:          sio.in,
				Implicits:       sio.implicitSrcs,
				Outputs:         sio.out,
				ImplicitOutputs: sio.implicitOuts,
				Args:            args,
				Depfile:         sio.depfile,
			})
	}
}

func (m *genBackend) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	args, implicits := m.getArgs(ctx)

	m.genDir = android.PathForModuleGen(ctx)
	m.exportGenIncludeDirs = pathsForModuleGen(ctx, m.Properties.Export_gen_include_dirs)

	if m.Properties.Tool != "" {
		tool := android.PathForSource(ctx, filepath.Join(ctx.ModuleDir(), m.Properties.Tool))
		args["tool"] = tool.String()
		implicits = append(implicits, tool)
	}

	if len(m.Properties.Out) > 0 {
		sio := soongInout{
			in:           android.PathsForSource(ctx, m.Properties.Srcs),
			implicitSrcs: implicits,
			out:          pathsForModuleGen(ctx, m.Properties.Out),
			implicitOuts: pathsForModuleGen(ctx, m.Properties.Implicit_outs),
		}
		if m.Properties.Depfile {
			sio.depfile = android.PathForModuleGen(ctx, getDepfileName(m.Name()))
		}

		m.inouts = append(m.inouts, sio)
	}

	re := regexp.MustCompile(m.Properties.TransformSourceProps.Out.Match)
	for _, tsrc := range m.Properties.Transform_srcs {
		srcPath := newSourceFilePath(tsrc, ctx, getBackend(ctx))
		io := m.Properties.inoutForSrc(re, srcPath, "", &m.Properties.Depfile)
		sio := soongInout{
			in:           android.PathsForSource(ctx, io.in),
			implicitSrcs: android.PathsForSource(ctx, io.implicitSrcs),
			out:          pathsForModuleGen(ctx, io.out),
			implicitOuts: pathsForModuleGen(ctx, io.implicitOuts),
		}
		if m.Properties.Depfile {
			sio.depfile = android.PathForModuleGen(ctx, io.depfile)
		}
		m.inouts = append(m.inouts, sio)
	}

	m.buildInouts(ctx, args)
}

func (gc *generateCommon) getHostBinModule(mctx android.TopDownMutatorContext) (hostBin android.Module) {
	mctx.VisitDirectDepsWithTag(hostToolBinTag, func(m android.Module) {
		hostBin = m
	})
	if hostBin == nil {
		panic(fmt.Errorf("Could not find module specified by `host_bin: %v`", proptools.String(gc.Properties.Host_bin)))
	}
	return
}

func (gc *generateCommon) getHostBinModuleName(mctx android.TopDownMutatorContext) string {
	if gc.Properties.Host_bin == nil {
		return ""
	}
	return ccModuleName(mctx, gc.getHostBinModule(mctx).Name())
}

func (gc *generateCommon) createGenrule(mctx android.TopDownMutatorContext,
	out, implicitSrcs, implicitOuts []string, depfile bool) {

	if !isEnabled(gc) {
		return
	}

	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(gc.Properties.Cmd), "${args}",
		strings.Join(gc.Properties.Args, " "), -1)

	nameProps := nameProps{
		proptools.StringPtr(gc.Name()),
	}

	genProps := genBackendProps{
		Srcs:                    gc.Properties.getSources(mctx),
		Out:                     out,
		Implicit_srcs:           implicitSrcs,
		Implicit_outs:           implicitOuts,
		Export_gen_include_dirs: gc.Properties.Export_gen_include_dirs,
		Tool:                    proptools.String(gc.Properties.Tool),
		HostBin:                 gc.getHostBinModuleName(mctx),
		Cmd:                     cmd,
		Depfile:                 depfile,
		Module_deps:             gc.Properties.Module_deps,
		Module_srcs:             gc.Properties.Module_srcs,
		Encapsulates:            gc.Properties.Encapsulates,
		Cflags:                  gc.Properties.FlagArgsBuild.Cflags,
		Conlyflags:              gc.Properties.FlagArgsBuild.Conlyflags,
		Cxxflags:                gc.Properties.FlagArgsBuild.Cxxflags,
		Asflags:                 gc.Properties.FlagArgsBuild.Asflags,
		Ldflags:                 gc.Properties.FlagArgsBuild.Ldflags,
		Ldlibs:                  gc.Properties.FlagArgsBuild.Ldlibs,
	}

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(android.ModuleFactoryAdaptor(genBackendFactory), &nameProps, &genProps)
}

func (gs *generateSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	gs.createGenrule(mctx, gs.Properties.Out, gs.Properties.Implicit_srcs, gs.Properties.Implicit_outs, proptools.Bool(gs.generateCommon.Properties.Depfile))
}

func (gs *generateStaticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	name := gs.Name()
	gs.createGenrule(mctx, []string{name + ".a"}, []string{}, []string{}, false)
}

func (gs *generateSharedLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	name := gs.Name()
	gs.createGenrule(mctx, []string{name + ".so"}, []string{}, []string{}, false)
}

func (gb *generateBinary) soongBuildActions(mctx android.TopDownMutatorContext) {
	name := gb.Name()
	gb.createGenrule(mctx, []string{name}, []string{}, []string{}, false)
}

var (
	// Use raw string literal backtick to avoid having to escape the
	// backslash in the regular expressions
	varRegexp = regexp.MustCompile(`\$[0-9]+`)
	dotRegexp = regexp.MustCompile(`\.{2,}`)
	extRegexp = regexp.MustCompile(`^\.`)
)

func (ts *transformSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	if !isEnabled(ts) {
		return
	}

	nameProps := nameProps{proptools.StringPtr(ts.Name())}

	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(ts.generateCommon.Properties.Cmd), "${args}",
		strings.Join(ts.generateCommon.Properties.Args, " "), -1)

	genProps := genBackendProps{
		Transform_srcs:          ts.generateCommon.Properties.getSources(mctx),
		Export_gen_include_dirs: ts.generateCommon.Properties.Export_gen_include_dirs,
		Tool:                    proptools.String(ts.generateCommon.Properties.Tool),
		HostBin:                 ts.getHostBinModuleName(mctx),
		Cmd:                     cmd,
		Depfile:                 proptools.Bool(ts.generateCommon.Properties.Depfile),
		Module_deps:             ts.generateCommon.Properties.Module_deps,
		Module_srcs:             ts.generateCommon.Properties.Module_srcs,
		Encapsulates:            ts.generateCommon.Properties.Encapsulates,
		TransformSourceProps:    ts.Properties.TransformSourceProps,
	}

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(android.ModuleFactoryAdaptor(genBackendFactory), &nameProps, &genProps)
}
