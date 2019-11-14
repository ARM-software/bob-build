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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"android/soong/android"
	"android/soong/genrule"

	"github.com/ARM-software/bob-build/utils"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type genBackendProps struct {
	Srcs                    []string
	Out                     []string
	Export_gen_include_dirs []string
	Cmd                     string
	HostBin                 string
	Tool                    string
	Depfile                 bool
	Module_deps             []string
	Module_srcs             []string
	Encapsulates            []string
}

type genBackend struct {
	android.ModuleBase
	Properties genBackendProps

	exportGenIncludeDirs android.WritablePaths
	inouts               []soongInout
}

func genBackendFactory() android.Module {
	m := &genBackend{}

	m.AddProperties(&m.Properties)
	android.InitAndroidModule(m)

	return m
}

func (m *genBackend) filterOutputs(predicate func(string) bool) (ret android.Paths) {
	for _, io := range m.inouts {
		for _, p := range io.out {
			if predicate(p.String()) {
				ret = append(ret, p)
			}
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

func (m *genBackend) getArgs(ctx android.ModuleContext) map[string]string {
	args := map[string]string{
		"bob_config":      filepath.Join(getBuildDir(), configName),
		"bob_config_opts": configOpts,
		"gen_dir":         android.PathForModuleGen(ctx).String(),
		"host_bin":        m.getHostBin(ctx),

		// flag_defaults is primarily used to invoke sub-makes of
		// different libraries. This shouldn't be needed on Android.
		// This means the following can't be expanded:
		"ar":         "",
		"as":         "",
		"asflags":    "",
		"cc":         "",
		"cflags":     "",
		"conlyflags": "",
		"cxx":        "",
		"cxxflags":   "",
		"ldflags":    "",
		"linker":     "",
	}
	// TODO: Support `${xxmod_out}`
	ctx.VisitDirectDepsIf(
		func(dep android.Module) bool {
			tag := ctx.OtherModuleDependencyTag(dep)
			if tag == generatedSourceTag || tag == generatedDepTag {
				return true
			}
			return false
		},
		func(dep android.Module) {
			args[dep.Name()+"_out"] = ""
		})
	return args
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
	args := m.getArgs(ctx)
	implicits := []android.Path{}

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
		}
		if m.Properties.Depfile {
			sio.depfile = android.PathForModuleGen(ctx, getDepfileName(m.Name()))
		}

		m.inouts = append(m.inouts, sio)
	}

	m.buildInouts(ctx, args)
}

func expandBobVariables(str, tool, hostBin string) (out string, err error) {
	// Bob is lax on whether there is any parenthesis, or whether
	// () or {} is used. That's because it relies on the expansion
	// happening in ninja. In a few cases Bob explicitly looks for
	// {}. Soong wants ()
	//
	// ${host_bin} => $(location tool) or $(location)
	// ${tool} => $(location tool_file) or $(location)
	// ${in} => $(in)
	// ${out} => $(out)
	// ${depfile} => $(depfile)
	// ${gen_dir} => $(genDir)
	//
	// ${bob_config} - expand inline
	// ${bob_config_opts} - expand inline
	// ${args} - expand inline
	//
	// {{match_srcs x}} => $(location x) ignoring globs
	//
	// ${src_dir}, ${module_dir}, ${xxmod_dir}, ${xxmod_outs} don't appear to be supported
	// Nor are ${srcs_generated}, ${headers_generated}
	//
	// flag_defaults is primarily used to invoke sub-makes of
	// different libraries. This shouldn't be needed on Android.
	// This means the following can't be expanded:
	//
	// ${ar}
	// ${as} ${asflags}
	// ${cc} ${cflags} ${conlyflags}
	// ${cxx} ${cxxflags}
	// ${linker} ${ldflags}
	out = os.Expand(str, func(s string) string {
		switch s {
		case "host_bin":
			if hostBin != "" {
				return "$(location " + hostBin + ")"
			} else {
				err = errors.New("${host_bin} used but host_bin not specified")
				return "$(location)"
			}
		case "tool":
			if tool != "" {
				return "$(location " + tool + ")"
			} else {
				err = errors.New("${tool} used but tool not specified")
				return "$(location)"
			}
		case "in":
			return "$(in)"
		case "out":
			return "$(out)"
		case "depfile":
			return "$(depfile)"
		case "gen_dir":
			return "$(genDir)"
		case "bob_config":
			return filepath.Join(getBuildDir(), configName)
		case "bob_config_opts":
			return configOpts
		default:
			return ""
		}
	})
	return
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

func (gc *generateCommon) createGenrule(mctx android.TopDownMutatorContext,
	out []string, depfile bool) {

	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(gc.Properties.Cmd), "${args}",
		strings.Join(gc.Properties.Args, " "), -1)

	hostBinModuleName := ""
	if gc.Properties.Host_bin != nil {
		hostBinModuleName = ccModuleName(mctx, gc.getHostBinModule(mctx).Name())
	}

	nameProps := nameProps{
		proptools.StringPtr(gc.Name()),
	}

	genProps := genBackendProps{
		Srcs:                    gc.Properties.getSources(mctx),
		Out:                     out,
		Export_gen_include_dirs: gc.Properties.Export_gen_include_dirs,
		Tool:                    proptools.String(gc.Properties.Tool),
		HostBin:                 hostBinModuleName,
		Cmd:                     cmd,
		Depfile:                 depfile,
		Module_deps:             gc.Properties.Module_deps,
		Module_srcs:             gc.Properties.Module_srcs,
		Encapsulates:            gc.Properties.Encapsulates,
	}

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(android.ModuleFactoryAdaptor(genBackendFactory), &nameProps, &genProps)
}

func (gs *generateSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	gs.createGenrule(mctx, gs.Properties.Out, proptools.Bool(gs.generateCommon.Properties.Depfile))
}

func (gs *generateStaticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	name := gs.Name()
	gs.createGenrule(mctx, []string{name + ".a"}, false)
}

func (gs *generateSharedLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	name := gs.Name()
	gs.createGenrule(mctx, []string{name + ".so"}, false)
}

func (gb *generateBinary) soongBuildActions(mctx android.TopDownMutatorContext) {
	name := gb.Name()
	gb.createGenrule(mctx, []string{name}, false)
}

var (
	// Use raw string literal backtick to avoid having to escape the
	// backslash in the regular expressions
	varRegexp = regexp.MustCompile(`\$[0-9]+`)
	dotRegexp = regexp.MustCompile(`\.{2,}`)
	extRegexp = regexp.MustCompile(`^\.`)
)

// Bob's module type allows the output file name to be specified using
// a regular expression replace, whereas Soong only allows you to
// specify the a new extension for the output.
//
// Look for an extension Bob's replacement string, and just use that. The
// replacement output filename won't be precisely as specified. If we
// want to maintain Bob behaviour we will need our own module type
// based on gensrcs.
func soongOutputExtension(re string) string {
	// We have a regular expression which might look like one of
	//  $1.ext
	//  $1.$2.ext
	//  $1.infix.$2.ext
	//  dir/$1.ext
	//
	// Drop the directory, match parts, and first '.', so we just end
	// up with `ext` or `infix.ext`. This should be good enough to keep
	// the files unique, and hopefully won't upset anything.
	// Drop directory
	dirChr := strings.LastIndex(re, "/")
	if dirChr > -1 {
		re = re[dirChr+1:]
	}
	// Remove capture group references, $[0-9]+
	re = varRegexp.ReplaceAllLiteralString(re, "")
	// With the capture group references removed, eliminate '.' which
	// are now adjacent by replacing .. with .
	// Note that this could cause breakage if we actually have '..'
	// (parent dir) in the replacement string, but that shouldn't be
	// happening.
	re = dotRegexp.ReplaceAllLiteralString(re, ".")
	// Trim initial '.'
	return extRegexp.ReplaceAllLiteralString(re, "")
}

func (ts *transformSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	// bob_transform_source maps best to gensrcs
	//
	// Setup a gensrcs property struct as if blueprint had read it
	// Only include the fields that we expect to use
	transformProps := struct {
		Name             *string
		Srcs             []string
		Exclude_srcs     []string
		Output_extension *string
		Depfile          *bool

		Tool_files []string
		Tools      []string
		Cmd        *string

		Export_include_dirs []string

		Owner       *string
		Proprietary *bool

		Enabled *bool
	}{}

	transformProps.Name = proptools.StringPtr(ts.Name())
	transformProps.Srcs = relativeToModuleDir(mctx, ts.generateCommon.Properties.Srcs)
	transformProps.Exclude_srcs = relativeToModuleDir(mctx, ts.generateCommon.Properties.Exclude_srcs)
	transformProps.Export_include_dirs = ts.generateCommon.Properties.Export_gen_include_dirs
	transformProps.Enabled = ts.generateCommon.Properties.Enabled

	// Only set Tool_files or Tool if the Bob property is not ""
	// otherwise Soong will report a missing dependency
	if ts.generateCommon.Properties.Tool != nil {
		transformProps.Tool_files = []string{proptools.String(ts.generateCommon.Properties.Tool)}
	}
	if ts.generateCommon.Properties.Host_bin != nil {
		transformProps.Tools = []string{proptools.String(ts.generateCommon.Properties.Host_bin)}
	}

	transformProps.Depfile = ts.generateCommon.Properties.Depfile

	if len(ts.Properties.Out.Replace) > 1 {
		panic(fmt.Errorf("Multiple outputs not supported in bob_transform_source on soong, %s", mctx.ModuleName()))
	}
	transformProps.Output_extension =
		proptools.StringPtr(soongOutputExtension(ts.Properties.Out.Replace[0]))

	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(ts.generateCommon.Properties.Cmd), "${args}",
		strings.Join(ts.generateCommon.Properties.Args, " "), -1)

	cmd2, err := expandBobVariables(cmd, proptools.String(ts.generateCommon.Properties.Tool),
		proptools.String(ts.generateCommon.Properties.Host_bin))
	if err != nil {
		panic(fmt.Errorf("%s property cmd %s", mctx.ModuleName(), err.Error()))
	}
	transformProps.Cmd = proptools.StringPtr(cmd2)

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(android.ModuleFactoryAdaptor(genrule.GenSrcsFactory), &transformProps)
}
