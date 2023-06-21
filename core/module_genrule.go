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
	"strings"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
)

/*
	We are swapping from bob_generate_source to bob_genrule

bob_genrule is made to be a stricter version that is compatible with Android.
For easiest compatibility, we are using Androids format for genrule.
Some properties in the struct may not be useful, but it is better to expose as many
features as possible rather than too few. Some are commented out as they would take special
implementation for features we do not already have in place.
*/
type AndroidGenerateRuleProps struct {
	Out         []string
	ResolvedOut file.Paths `blueprint:"mutated"`
}

type AndroidGenerateCommonProps struct {
	// See https://ci.android.com/builds/submitted/8928481/linux/latest/view/soong_build.html
	Name                string
	Srcs                []string // TODO: This module should probalby make use of LegacySourceProps
	Exclude_srcs        []string
	Cmd                 *string
	Depfile             *bool
	Export_include_dirs []string
	Tool_files          []string
	Tools               []string

	ResolvedSrcs file.Paths `blueprint:"mutated"` // Glob results.
}

type AndroidGenerateCommonPropsInterface interface {
	pathProcessor
	FileConsumer
	FileResolver
}

var _ AndroidGenerateCommonPropsInterface = (*AndroidGenerateCommonProps)(nil) // impl check

func (ag *AndroidGenerateCommonProps) processPaths(ctx blueprint.BaseModuleContext) {

	prefix := projectModuleDir(ctx)
	// We don't want to process module dependencies as paths, we must filter them out first.

	srcs := utils.MixedListToFiles(ag.Srcs)
	targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Srcs), ":")

	ag.Srcs = append(utils.PrefixDirs(srcs, prefix), targets...)
	ag.Exclude_srcs = utils.PrefixDirs(ag.Exclude_srcs, prefix)

	tool_files_targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Tool_files), ":")
	ag.Tool_files = utils.PrefixDirs(utils.MixedListToFiles(ag.Tool_files), prefix)
	ag.Tool_files = append(ag.Tool_files, tool_files_targets...)

	// When we specify a specific tag, its location will be incorrect as we move everything into a top level bp,
	// we must fix this by iterating through the command.
	matches := locationTagRegex.FindAllStringSubmatch(*ag.Cmd, -1)
	for _, v := range matches {
		tag := v[1]
		if tag[0] == ':' {
			continue
		}
		newTag := utils.PrefixDirs([]string{tag}, prefix)[0]
		// Replacing with space allows us to not replace the same basename more than once if it appears
		// multiple times.
		newCmd := strings.Replace(*ag.Cmd, " "+tag, " "+newTag, -1)
		ag.Cmd = &newCmd
	}
}

func (ag *AndroidGenerateCommonProps) ResolveFiles(ctx blueprint.BaseModuleContext) {
	// Since globbing is supported we must call a resolver.
	files := file.Paths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(ag.Srcs), ag.Exclude_srcs) {
		fp := file.NewPath(match, ctx.ModuleName(), file.TypeUnset)
		files = files.AppendIfUnique(fp)
	}

	ag.ResolvedSrcs = files
}

func (ag *AndroidGenerateCommonProps) GetTargets() []string {
	return utils.MixedListToBobTargets(ag.Srcs)
}

func (ag *AndroidGenerateCommonProps) GetDirectFiles() file.Paths {
	return ag.ResolvedSrcs
}

func (ag *AndroidGenerateCommonProps) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return ag.GetDirectFiles().Merge(ReferenceGetFilesImpl(ctx))
}

type ModuleGenruleCommon struct {
	module.ModuleBase
	simpleOutputProducer
	headerProducer
	Properties struct {
		EnableableProps
		Features
		AndroidGenerateCommonProps
	}
	deps []string
}

var _ FileConsumer = (*ModuleGenruleCommon)(nil)

func (m *ModuleGenruleCommon) outputs() []string {
	return m.outs
}

func (m *ModuleGenruleCommon) init(properties *config.Properties, list ...interface{}) {
	m.Properties.Features.Init(properties, list...)
}

func (m *ModuleGenruleCommon) processPaths(ctx blueprint.BaseModuleContext) {
	m.deps = utils.MixedListToBobTargets(m.Properties.AndroidGenerateCommonProps.Tool_files)
	m.Properties.AndroidGenerateCommonProps.processPaths(ctx)
}

func (m *ModuleGenruleCommon) GetTargets() []string {
	return m.Properties.GetTargets()
}

func (m *ModuleGenruleCommon) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.GetFiles(ctx)
}

func (m *ModuleGenruleCommon) GetDirectFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleGenruleCommon) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleGenruleCommon) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleGenruleCommon) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.EnableableProps, &m.Properties.AndroidGenerateCommonProps}
}

func (m *ModuleGenruleCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getAndroidGenerateCommonInterface interface {
	getAndroidGenerateCommon() *ModuleGenruleCommon
}

func (m *ModuleGenruleCommon) getAndroidGenerateCommon() *ModuleGenruleCommon {
	return m
}

func getAndroidGenerateCommon(i interface{}) (*ModuleGenruleCommon, bool) {
	var gsc *ModuleGenruleCommon
	gsd, ok := i.(getAndroidGenerateCommonInterface)
	if ok {
		gsc = gsd.getAndroidGenerateCommon()
	}
	return gsc, ok
}

type ModuleGenrule struct {
	ModuleGenruleCommon
	Properties struct {
		AndroidGenerateRuleProps
	}
}

type androidGenerateRuleInterface interface {
	FileConsumer
	FileResolver
	pathProcessor
}

var _ androidGenerateRuleInterface = (*ModuleGenrule)(nil) // impl check

func (m *ModuleGenrule) processPaths(ctx blueprint.BaseModuleContext) {
	m.ModuleGenruleCommon.processPaths(ctx)
}

func (m *ModuleGenrule) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.ModuleGenruleCommon.ResolveFiles(ctx)

	files := file.Paths{}
	for _, out := range m.Properties.Out {
		fp := file.NewPath(out, ctx.ModuleName(), file.TypeGenerated)
		files = files.AppendIfUnique(fp)
	}

	m.Properties.ResolvedOut = files
}

func (m *ModuleGenrule) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.ModuleGenruleCommon.Properties.GetFiles(ctx)
}

func (m *ModuleGenrule) GetDirectFiles() file.Paths {
	return m.ModuleGenruleCommon.Properties.GetDirectFiles()
}

func (m *ModuleGenrule) GetTargets() []string {
	return m.ModuleGenruleCommon.Properties.GetTargets()
}

func (m *ModuleGenrule) OutFiles() file.Paths {
	return m.Properties.ResolvedOut
}

func (m *ModuleGenrule) OutFileTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleGenrule) FlagsOut() (flags flag.Flags) {
	gc, _ := getAndroidGenerateCommon(m)
	for _, str := range gc.Properties.Export_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

func (m *ModuleGenrule) shortName() string {
	return m.Name()
}

func (m *ModuleGenrule) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getGenerator(ctx)
		g.genruleActions(m, ctx)
	}
}

func (m ModuleGenrule) GetProperties() interface{} {
	return m.Properties
}

func (m *ModuleGenrule) FeaturableProperties() []interface{} {
	return append(m.ModuleGenruleCommon.FeaturableProperties(), &m.Properties.AndroidGenerateRuleProps)
}

func generateRuleAndroidFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGenrule{}

	module.ModuleGenruleCommon.init(&config.Properties,
		AndroidGenerateCommonProps{}, AndroidGenerateRuleProps{}, EnableableProps{})

	return module, []interface{}{&module.ModuleGenruleCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
