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
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

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

	ResolvedOut file.Paths `blueprint:"mutated"`
}

type ModuleGenerateSource struct {
	ModuleGenerateCommon
	Properties struct {
		GenerateSourceProps
	}
}

type generateSourceInterface interface {
	installable
	pathProcessor
	FileResolver
	FileProvider
	FileConsumer
}

var _ generateSourceInterface = (*ModuleGenerateSource)(nil) // impl check

func (m *ModuleGenerateSource) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool {
			// TODO: Consider adding a better group tag
			return f.IsNotType(file.TypeRsp) &&
				f.IsNotType(file.TypeDep)
		},
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleGenerateSource) implicitOutputs() []string {
	return []string{}
}

func (m *ModuleGenerateSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).generateSourceActions(m, ctx)
	}
}

func (m *ModuleGenerateSource) FeaturableProperties() []interface{} {
	return append(m.ModuleGenerateCommon.FeaturableProperties(), &m.Properties.GenerateSourceProps)
}

func (m *ModuleGenerateSource) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.Implicit_srcs = utils.PrefixDirs(m.Properties.Implicit_srcs, projectModuleDir(ctx))
	m.Properties.Exclude_implicit_srcs = utils.PrefixDirs(m.Properties.Exclude_implicit_srcs, projectModuleDir(ctx))
	m.ModuleGenerateCommon.processPaths(ctx)
}

func (m *ModuleGenerateSource) ResolveFiles(ctx blueprint.BaseModuleContext) {
	// Resolve sources.
	gc, _ := getGenerateCommon(m)
	gc.Properties.LegacySourceProps.ResolveFiles(ctx)

	// Resolve output files
	outs := file.Paths{}
	for _, out := range m.Properties.Out {
		fp := file.NewPath(out, ctx.ModuleName(), file.TypeGenerated)
		outs = outs.AppendIfUnique(fp)
	}

	for _, implicit := range glob(ctx, m.Properties.Implicit_srcs, m.Properties.Exclude_implicit_srcs) {
		fp := file.NewPath(implicit, ctx.ModuleName(), file.TypeImplicit)
		gc.Properties.LegacySourceProps.ResolvedSrcs = gc.Properties.LegacySourceProps.ResolvedSrcs.AppendIfUnique(fp)
	}

	m.Properties.ResolvedOut = outs

}

func (m *ModuleGenerateSource) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.LegacySourceProps.GetFiles(ctx)
}

func (m *ModuleGenerateSource) GetDirectFiles() file.Paths {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.LegacySourceProps.GetDirectFiles()
}

func (m *ModuleGenerateSource) GetTargets() (tgts []string) {
	gc, _ := getGenerateCommon(m)
	tgts = append(tgts, gc.Properties.LegacySourceProps.GetTargets()...)
	tgts = append(tgts, gc.Properties.Generated_sources...)
	return
}

func (m *ModuleGenerateSource) OutFiles() file.Paths {
	gc, _ := getGenerateCommon(m)
	return append(m.Properties.ResolvedOut, gc.OutFiles()...)
}

func (m *ModuleGenerateSource) OutFileTargets() []string {
	return []string{}
}

func (m *ModuleGenerateSource) FlagsOut() (flags flag.Flags) {
	gc, _ := getGenerateCommon(m)
	for _, str := range gc.Properties.Export_gen_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

// Return an inouts structure naming all the files associated with a
// generateSource's inputs.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies
// to out, depfile and rspfile. The output directory (if needed) needs to be
// added in by the backend specific GenerateBuildAction()
func (m *ModuleGenerateSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var io inout

	m.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			if fp.IsType(file.TypeImplicit) {
				io.implicitSrcs = append(io.implicitSrcs, fp.BuildPath())
			} else {
				io.in = append(io.in, fp.BuildPath())
			}
			return true
		})

	io.out = m.Properties.Out

	if depfile, ok := m.OutFiles().FindSingle(
		func(p file.Path) bool { return p.IsType(file.TypeDep) }); ok {
		io.depfile = depfile.UnScopedPath()
	}

	if rspfile, ok := m.OutFiles().FindSingle(
		func(p file.Path) bool { return p.IsType(file.TypeRsp) }); ok {
		io.rspfile = rspfile.UnScopedPath()
	}

	return []inout{io}
}

func (m *ModuleGenerateSource) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	// Install everything that we generate
	return m.outputs()
}

func (m ModuleGenerateSource) GetProperties() interface{} {
	return m.Properties
}

func generateSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGenerateSource{}
	module.ModuleGenerateCommon.init(&config.Properties,
		GenerateProps{}, GenerateSourceProps{})

	return module, []interface{}{&module.ModuleGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
