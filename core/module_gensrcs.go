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
)

type GensrcsRuleProps struct {
	Output_extension string
	ResolvedOut      file.Paths `blueprint:"mutated"`
}

type ModuleGensrcs struct {
	ModuleGenruleCommon
	Properties struct {
		GensrcsRuleProps
	}
}

func (m *ModuleGensrcs) processPaths(ctx blueprint.BaseModuleContext) {
	m.ModuleGenruleCommon.processPaths(ctx)

	prefix := projectModuleDir(ctx)

	m.ModuleGenruleCommon.Properties.Export_include_dirs = utils.PrefixDirs(m.ModuleGenruleCommon.Properties.Export_include_dirs, prefix)
}

func (m *ModuleGensrcs) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.ModuleGenruleCommon.ResolveFiles(ctx)

	files := file.Paths{}
	for _, out := range m.ModuleGenruleCommon.Properties.Srcs {
		// replace extension
		fp := file.NewPath(pathtools.ReplaceExtension(out, m.Properties.Output_extension), ctx.ModuleName(), file.TypeGenerated)
		files = files.AppendIfUnique(fp)
	}

	m.Properties.ResolvedOut = files
}

func (m *ModuleGensrcs) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.ModuleGenruleCommon.Properties.GetFiles(ctx)
}

func (m *ModuleGensrcs) GetDirectFiles() file.Paths {
	return m.ModuleGenruleCommon.Properties.GetDirectFiles()
}

func (m *ModuleGensrcs) GetTargets() []string {
	return m.ModuleGenruleCommon.Properties.GetTargets()
}

func (m *ModuleGensrcs) OutFiles() file.Paths {
	return m.Properties.ResolvedOut
}

func (m *ModuleGensrcs) OutFileTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleGensrcs) shortName() string {
	return m.Name()
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
	gc, _ := getAndroidGenerateCommon(m)
	for _, str := range gc.Properties.Export_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

func (m *ModuleGensrcs) FeaturableProperties() []interface{} {
	return append(m.ModuleGenruleCommon.FeaturableProperties(), &m.Properties.GensrcsRuleProps)
}

func gensrcsFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGensrcs{}

	module.ModuleGenruleCommon.init(&config.Properties,
		AndroidGenerateCommonProps{}, GensrcsRuleProps{}, EnableableProps{})

	return module, []interface{}{&module.ModuleGenruleCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
