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
	"regexp"
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
type GenruleProps struct {
	Out         []string
	ResolvedOut file.Paths `blueprint:"mutated"`
}

type StrictGenerateProps struct {
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

type StrictGeneratePropsInterface interface {
	pathProcessor
	FileConsumer
	FileResolver
}

var variableRegex = regexp.MustCompile(`\$\(([A-Za-z- \._:0-9]+)\)`)
var _ StrictGeneratePropsInterface = (*StrictGenerateProps)(nil) // impl check

func (ag *StrictGenerateProps) processPaths(ctx blueprint.BaseModuleContext) {

	prefix := projectModuleDir(ctx)
	// We don't want to process module dependencies as paths, we must filter them out first.

	srcs := utils.MixedListToFiles(ag.Srcs)
	targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Srcs), ":")

	ag.Srcs = append(utils.PrefixDirs(srcs, prefix), targets...)
	ag.Exclude_srcs = utils.PrefixDirs(ag.Exclude_srcs, prefix)

	ag.validateCmd(ctx)

	// When we specify a specific tag, its location will be incorrect as we move everything into a top level bp,
	// we must fix this by iterating through the command.
	matches := locationTagRegex.FindAllStringSubmatch(*ag.Cmd, -1)
	for _, v := range matches {
		tag := v[1]
		if tag[0] == ':' {
			continue
		}

		// do not prefix paths for `Tools` which are host binary modules
		if utils.Contains(ag.Tool_files, tag) {
			newTag := utils.PrefixDirs([]string{tag}, prefix)[0]
			// Replacing with space allows us to not replace the same basename more than once if it appears
			// multiple times.
			newCmd := strings.Replace(*ag.Cmd, " "+tag, " "+newTag, -1)
			ag.Cmd = &newCmd
		}
	}

	tool_files_targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Tool_files), ":")
	ag.Tool_files = utils.PrefixDirs(utils.MixedListToFiles(ag.Tool_files), prefix)
	ag.Tool_files = append(ag.Tool_files, tool_files_targets...)
}

func (ag *StrictGenerateProps) validateCmd(ctx blueprint.BaseModuleContext) {

	// for variables only curly brackets are allowed
	matches := variableRegex.FindAllStringSubmatch(*ag.Cmd, -1)

	for _, v := range matches {
		ctx.ModuleErrorf("Only curly brackets are allowed in `cmd`. Use: '${%s}'", v[1])
	}

	// Check default tool
	if strings.Contains(*ag.Cmd, "${location}") {
		if len(ag.Tools) > 0 && len(ag.Tool_files) > 0 {
			ctx.ModuleErrorf("You cannot have default $(location) specified in `cmd` if setting both `tool_files` and `tools`.")
		}
	}
}

func (ag *StrictGenerateProps) ResolveFiles(ctx blueprint.BaseModuleContext) {
	// Since globbing is supported we must call a resolver.
	files := file.Paths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(ag.Srcs), ag.Exclude_srcs) {
		fp := file.NewPath(match, ctx.ModuleName(), file.TypeUnset)
		files = files.AppendIfUnique(fp)
	}

	ag.ResolvedSrcs = files
}

func (ag *StrictGenerateProps) GetTargets() []string {
	return utils.MixedListToBobTargets(ag.Srcs)
}

func (ag *StrictGenerateProps) GetDirectFiles() file.Paths {
	return ag.ResolvedSrcs
}

func (ag *StrictGenerateProps) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return ag.GetDirectFiles().Merge(ReferenceGetFilesImpl(ctx))
}

type ModuleStrictGenerateCommon struct {
	module.ModuleBase
	Properties struct {
		EnableableProps
		Features
		StrictGenerateProps
	}
	deps []string
}

var _ FileConsumer = (*ModuleStrictGenerateCommon)(nil)

func (m *ModuleStrictGenerateCommon) init(properties *config.Properties, list ...interface{}) {
	m.Properties.Features.Init(properties, list...)
}

func (m *ModuleStrictGenerateCommon) processPaths(ctx blueprint.BaseModuleContext) {
	m.deps = utils.MixedListToBobTargets(m.Properties.StrictGenerateProps.Tool_files)
	m.Properties.StrictGenerateProps.processPaths(ctx)
}

func (m *ModuleStrictGenerateCommon) GetTargets() []string {
	return m.Properties.GetTargets()
}

func (m *ModuleStrictGenerateCommon) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.GetFiles(ctx)
}

func (m *ModuleStrictGenerateCommon) GetDirectFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleStrictGenerateCommon) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleStrictGenerateCommon) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleStrictGenerateCommon) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.EnableableProps, &m.Properties.StrictGenerateProps}
}

func (m *ModuleStrictGenerateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

// Module implementing getGenerateCommonInterface are able to generate output files
type getAndroidGenerateCommonInterface interface {
	getAndroidGenerateCommon() *ModuleStrictGenerateCommon
}

func (m *ModuleStrictGenerateCommon) getAndroidGenerateCommon() *ModuleStrictGenerateCommon {
	return m
}

func getAndroidGenerateCommon(i interface{}) (*ModuleStrictGenerateCommon, bool) {
	var gsc *ModuleStrictGenerateCommon
	gsd, ok := i.(getAndroidGenerateCommonInterface)
	if ok {
		gsc = gsd.getAndroidGenerateCommon()
	}
	return gsc, ok
}

type ModuleGenrule struct {
	ModuleStrictGenerateCommon
	Properties struct {
		GenruleProps
	}
}

type ModuleGenruleInterface interface {
	FileConsumer
	FileResolver
	pathProcessor
}

var _ ModuleGenruleInterface = (*ModuleGenrule)(nil) // impl check

func (m *ModuleGenrule) implicitOutputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsType(file.TypeImplicit) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleGenrule) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsNotType(file.TypeImplicit) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleGenrule) processPaths(ctx blueprint.BaseModuleContext) {
	m.ModuleStrictGenerateCommon.processPaths(ctx)
}

func (m *ModuleGenrule) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.ModuleStrictGenerateCommon.ResolveFiles(ctx)

	files := file.Paths{}
	for _, out := range m.Properties.Out {
		fp := file.NewPath(out, ctx.ModuleName(), file.TypeGenerated)
		files = files.AppendIfUnique(fp)
	}

	m.Properties.ResolvedOut = files
}

func (m *ModuleGenrule) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.ModuleStrictGenerateCommon.Properties.GetFiles(ctx)
}

func (m *ModuleGenrule) GetDirectFiles() file.Paths {
	return m.ModuleStrictGenerateCommon.Properties.GetDirectFiles()
}

func (m *ModuleGenrule) GetTargets() []string {
	return m.ModuleStrictGenerateCommon.Properties.GetTargets()
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

func (m *ModuleGenrule) generateInouts(ctx blueprint.ModuleContext) []inout {
	var io inout

	m.GetFiles(ctx).ForEachIf(
		// TODO: The current generator does pass parse .toc files when consuming generated shared libraries.
		func(fp file.Path) bool { return fp.IsNotType(file.TypeToc) },
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

	return []inout{io}
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
	return append(m.ModuleStrictGenerateCommon.FeaturableProperties(), &m.Properties.GenruleProps)
}

func generateRuleAndroidFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGenrule{}

	module.ModuleStrictGenerateCommon.init(&config.Properties,
		StrictGenerateProps{}, GenruleProps{}, EnableableProps{})

	return module, []interface{}{&module.ModuleStrictGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
