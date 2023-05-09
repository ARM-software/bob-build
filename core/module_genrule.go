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
	ResolvedOut FilePaths `blueprint:"mutated"`
}

type AndroidGenerateCommonProps struct {
	// See https://ci.android.com/builds/submitted/8928481/linux/latest/view/soong_build.html
	Name                string
	Srcs                []string // TODO: This module should probalby make use of LegacySourceProps
	Exclude_srcs        []string
	Cmd                 *string
	Depfile             *bool
	Enabled             *bool
	Export_include_dirs []string
	Tool_files          []string
	Tools               []string

	ResolvedSrcs FilePaths `blueprint:"mutated"` // Glob results.
}

type AndroidGenerateCommonPropsInterface interface {
	pathProcessor
	SourceFileConsumer
	FileResolver
}

var _ AndroidGenerateCommonPropsInterface = (*AndroidGenerateCommonProps)(nil) // impl check

func (ag *AndroidGenerateCommonProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {

	prefix := projectModuleDir(ctx)
	// We don't want to process module dependencies as paths, we must filter them out first.

	srcs := utils.MixedListToFiles(ag.Srcs)
	targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Srcs), ":")

	ag.Srcs = append(utils.PrefixDirs(srcs, prefix), targets...)
	ag.Exclude_srcs = utils.PrefixDirs(ag.Exclude_srcs, prefix)
	ag.Tool_files = utils.PrefixDirs(ag.Tool_files, prefix)

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

func (ag *AndroidGenerateCommonProps) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	// Since globbing is supported we must call a resolver.
	files := FilePaths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(ag.Srcs), ag.Exclude_srcs) {
		fp := newSourceFilePath(match, ctx, g)
		files = files.AppendIfUnique(fp)
	}

	ag.ResolvedSrcs = files
}

func (ag *AndroidGenerateCommonProps) GetSrcTargets() []string {
	return utils.MixedListToBobTargets(ag.Srcs)
}

func (ag *AndroidGenerateCommonProps) GetDirectSrcs() FilePaths {
	return ag.ResolvedSrcs
}

func (ag *AndroidGenerateCommonProps) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	return ag.GetDirectSrcs().Merge(ReferenceGetSrcsImpl(ctx))
}

type ModuleGenruleCommon struct {
	moduleBase
	EnableableProps
	simpleOutputProducer
	headerProducer
	Properties struct {
		AndroidGenerateCommonProps
	}
}

var _ SourceFileConsumer = (*ModuleGenruleCommon)(nil)

func (m *ModuleGenruleCommon) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.AndroidGenerateCommonProps.processPaths(ctx, g)
}

func (m *ModuleGenruleCommon) GetSrcTargets() []string {
	return m.Properties.GetSrcTargets()
}

func (m *ModuleGenruleCommon) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	return m.Properties.GetSrcs(ctx)
}

func (m *ModuleGenruleCommon) GetDirectSrcs() FilePaths {
	return m.Properties.GetDirectSrcs()
}

func (m *ModuleGenruleCommon) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.ResolveFiles(ctx, g)
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
	SourceFileConsumer
	FileResolver
	pathProcessor
}

var _ androidGenerateRuleInterface = (*ModuleGenrule)(nil) // impl check

func (m *ModuleGenrule) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.ModuleGenruleCommon.processPaths(ctx, g)
}

func (m *ModuleGenrule) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.ModuleGenruleCommon.ResolveFiles(ctx, g)

	files := FilePaths{}
	for _, out := range m.Properties.Out {
		fp := newGeneratedFilePathFromModule(out, ctx, g)
		files = files.AppendIfUnique(fp)
	}

	m.Properties.ResolvedOut = files
}

func (m *ModuleGenrule) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	return m.ModuleGenruleCommon.Properties.GetSrcs(ctx)
}

func (m *ModuleGenrule) GetDirectSrcs() FilePaths {
	return m.ModuleGenruleCommon.Properties.GetDirectSrcs()
}

func (m *ModuleGenrule) GetSrcTargets() []string {
	return m.ModuleGenruleCommon.Properties.GetSrcTargets()
}

func (m *ModuleGenrule) OutSrcs() FilePaths {
	return m.Properties.ResolvedOut
}

func (m *ModuleGenrule) OutSrcTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleGenrule) shortName() string {
	return m.Name()
}

func (m *ModuleGenrule) getEnableableProps() *EnableableProps {
	return &m.EnableableProps
}

func (m *ModuleGenrule) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.androidGenerateRuleActions(m, ctx)
	}
}

func (m ModuleGenrule) GetProperties() interface{} {
	return m.Properties
}

func generateRuleAndroidFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGenrule{}

	return module, []interface{}{&module.ModuleGenruleCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
