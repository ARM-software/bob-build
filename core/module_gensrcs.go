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
	"github.com/google/blueprint/pathtools"
	"github.com/google/blueprint/proptools"
)

/*
	We are swapping from `bob_transform_source` to `bob_gensrcs`

`bob_gensrcs` is made to be a stricter version that is compatible with Android.
For easiest compatibility, we are using Androids format for `gensrcs`.
Some properties in the struct may not be useful, but it is better to expose as many
features as possible rather than too few. Some are commented out as they would take special
implementation for features we do not already have in place.

*/

type GensrcsProps struct {
	Output_extension string
	ResolvedOut      file.Paths `blueprint:"mutated"`
}

type ModuleGensrcs struct {
	ModuleStrictGenerateCommon
	Properties struct {
		GensrcsProps
	}
}

func (m *ModuleGensrcs) implicitOutputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsType(file.TypeImplicit) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleGensrcs) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsNotType(file.TypeImplicit) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleGensrcs) processPaths(ctx blueprint.BaseModuleContext) {
	m.ModuleStrictGenerateCommon.processPaths(ctx)

	prefix := projectModuleDir(ctx)

	m.ModuleStrictGenerateCommon.Properties.Export_include_dirs = utils.PrefixDirs(m.ModuleStrictGenerateCommon.Properties.Export_include_dirs, prefix)
}

func (m *ModuleGensrcs) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.ModuleStrictGenerateCommon.ResolveFiles(ctx)
}

func (m *ModuleGensrcs) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.ModuleStrictGenerateCommon.Properties.GetFiles(ctx)
}

func (m *ModuleGensrcs) GetDirectFiles() file.Paths {
	return m.ModuleStrictGenerateCommon.Properties.GetDirectFiles()
}

func (m *ModuleGensrcs) GetTargets() []string {
	return m.ModuleStrictGenerateCommon.Properties.GetTargets()
}

func (m *ModuleGensrcs) OutFiles() file.Paths {
	return m.Properties.ResolvedOut
}

func (m *ModuleGensrcs) OutFileTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleGensrcs) ResolveOutFiles(ctx blueprint.BaseModuleContext) {
	files := file.Paths{}

	m.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			fpOut := file.NewPath(pathtools.ReplaceExtension(fp.ScopedPath(), m.Properties.Output_extension), ctx.ModuleName(), file.TypeGenerated)
			files = files.AppendIfUnique(fpOut)
			return true
		})

	m.Properties.ResolvedOut = files
}

func (m *ModuleGensrcs) shortName() string {
	return m.Name()
}

func (m *ModuleGensrcs) generateInouts(ctx blueprint.ModuleContext) []inout {
	var inouts []inout

	m.GetFiles(ctx).ForEachIf(
		func(fp file.Path) bool { return fp.IsNotType(file.TypeToc) },
		func(fp file.Path) bool {
			var io inout

			io.in = []string{fp.BuildPath()}
			io.out = []string{pathtools.ReplaceExtension(fp.ScopedPath(), m.Properties.Output_extension)}

			// TODO: check depfile
			if proptools.Bool(m.ModuleStrictGenerateCommon.Properties.Depfile) {
				io.depfile = getDepfileName(fp.UnScopedPath())
			}

			inouts = append(inouts, io)

			return true
		})

	return inouts
}

func (m *ModuleGensrcs) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getGenerator(ctx)
		g.gensrcsActions(m, ctx)
	}
}

func (m ModuleGensrcs) GetProperties() interface{} {
	return m.Properties
}

func (m *ModuleGensrcs) FlagsOut() (flags flag.Flags) {
	gc := m.getStrictGenerateCommon()
	for _, str := range gc.Properties.Export_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

func (m *ModuleGensrcs) FeaturableProperties() []interface{} {
	return append(m.ModuleStrictGenerateCommon.FeaturableProperties(), &m.Properties.GensrcsProps)
}

func (m *ModuleGensrcs) getStrictGenerateCommon() *ModuleStrictGenerateCommon {
	return &m.ModuleStrictGenerateCommon
}

func gensrcsFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGensrcs{}

	module.ModuleStrictGenerateCommon.init(&config.Properties,
		StrictGenerateProps{}, GensrcsProps{}, EnableableProps{})

	return module, []interface{}{&module.ModuleStrictGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
