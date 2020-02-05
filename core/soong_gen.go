// +build soong

/*
 * Copyright 2019-2020 Arm Limited.
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
	"strings"

	"android/soong/android"

	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/plugins/genrulebob"
)

func (gc *generateCommon) getSoongHostBinModule(mctx android.TopDownMutatorContext) *binary {
	var hostBinModule android.Module
	mctx.VisitDirectDepsWithTag(hostToolBinTag, func(m android.Module) {
		hostBinModule = m
	})
	if hostBinModule == nil {
		panic(fmt.Errorf("Could not find module specified by `host_bin: %v`", proptools.String(gc.Properties.Host_bin)))
	}
	bin, ok := hostBinModule.(*binary)
	if !ok {
		panic(fmt.Errorf("Host binary %s of module %s is not a bob_binary!", hostBinModule.Name(), gc.Name()))
	}
	return bin
}

func (gc *generateCommon) getHostBinModuleName(mctx android.TopDownMutatorContext) string {
	if gc.Properties.Host_bin == nil {
		return ""
	}
	return ccModuleName(mctx, gc.getSoongHostBinModule(mctx).Name())
}

func (gc *generateCommon) createGenrule(mctx android.TopDownMutatorContext,
	out, implicitSrcs, implicitOuts []string, depfile bool, factory func() android.Module) {

	if !isEnabled(gc) {
		return
	}

	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(gc.Properties.Cmd), "${args}",
		strings.Join(gc.Properties.Args, " "), -1)

	nameProps := nameProps{
		proptools.StringPtr(gc.Name()),
	}

	var tool string = ""
	if gc.Properties.Tool != nil {
		tool = relativeToModuleDir(mctx, []string{*gc.Properties.Tool})[0]
	}

	genProps := genrulebob.GenruleProps{
		Srcs:                    relativeToModuleDir(mctx, gc.Properties.getSources(mctx)),
		Out:                     out,
		Implicit_srcs:           relativeToModuleDir(mctx, implicitSrcs),
		Implicit_outs:           implicitOuts,
		Export_gen_include_dirs: gc.Properties.Export_gen_include_dirs,
		Tool:                    tool,
		Host_bin:                gc.getHostBinModuleName(mctx),
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
		Rsp_content:             gc.Properties.Rsp_content,
	}

	mctx.CreateModule(factory, &nameProps, &genProps)
}

func (gs *generateSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	gs.createGenrule(mctx, gs.Properties.Out, gs.Properties.getImplicitSources(mctx),
		gs.Properties.Implicit_outs, proptools.Bool(gs.generateCommon.Properties.Depfile),
		genrulebob.GenruleFactory)
}

func (gs *generateStaticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	if isEnabled(gs) {
		panic(fmt.Errorf("Generated static libraries are not supported"))
	}
}

func (gs *generateSharedLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	if isEnabled(gs) {
		panic(fmt.Errorf("Generated shared libraries are not supported"))
	}
}

func (gb *generateBinary) soongBuildActions(mctx android.TopDownMutatorContext) {
	if isEnabled(gb) {
		panic(fmt.Errorf("Generated binaries are not supported"))
	}
}

func (ts *transformSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	if !isEnabled(ts) {
		return
	}

	nameProps := nameProps{proptools.StringPtr(ts.Name())}

	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(ts.generateCommon.Properties.Cmd), "${args}",
		strings.Join(ts.generateCommon.Properties.Args, " "), -1)

	var tool string = ""
	if ts.generateCommon.Properties.Tool != nil {
		tool = relativeToModuleDir(mctx, []string{*ts.generateCommon.Properties.Tool})[0]
	}

	genProps := genrulebob.GenruleProps{
		Multi_out_srcs:          relativeToModuleDir(mctx, ts.generateCommon.Properties.getSources(mctx)),
		Export_gen_include_dirs: ts.generateCommon.Properties.Export_gen_include_dirs,
		Tool:                    tool,
		Host_bin:                ts.getHostBinModuleName(mctx),
		Cmd:                     cmd,
		Depfile:                 proptools.Bool(ts.generateCommon.Properties.Depfile),
		Module_deps:             ts.generateCommon.Properties.Module_deps,
		Module_srcs:             ts.generateCommon.Properties.Module_srcs,
		Encapsulates:            ts.generateCommon.Properties.Encapsulates,
		Multi_out_props: genrulebob.MultiOutProps{
			Match:         ts.Properties.TransformSourceProps.Out.Match,
			Replace:       ts.Properties.TransformSourceProps.Out.Replace,
			Implicit_srcs: ts.Properties.TransformSourceProps.Out.Implicit_srcs,
			Implicit_outs: ts.Properties.TransformSourceProps.Out.Implicit_outs,
		},
		Rsp_content: ts.generateCommon.Properties.Rsp_content,
	}

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(genrulebob.GenruleFactory, &nameProps, &genProps)
}
