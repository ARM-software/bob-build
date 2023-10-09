package core

import (
	"regexp"
	"strings"

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

type ModuleGensrcsInterface interface {
	file.Consumer
	file.Resolver
	pathProcessor
	Tagable
}

var _ ModuleGensrcsInterface = (*ModuleGensrcs)(nil) // impl check

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

			if proptools.Bool(m.ModuleStrictGenerateCommon.Properties.Depfile) {
				depOut := file.NewPath(fp.ScopedPath()+".d", ctx.ModuleName(), file.TypeDep|file.TypeGenerated|file.TypeImplicit)
				files = files.AppendIfUnique(depOut)
			}

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

			if depfile, ok := m.OutFiles().FindSingle(
				func(p file.Path) bool {
					return p.IsType(file.TypeDep) && strings.HasSuffix(p.UnScopedPath(), fp.UnScopedPath()+".d")
				}); ok {
				io.depfile = depfile.UnScopedPath()
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

func (m *ModuleGensrcs) HasTagRegex(query *regexp.Regexp) bool {
	return m.ModuleStrictGenerateCommon.HasTagRegex(query)
}

func (m *ModuleGensrcs) HasTag(query string) bool {
	return m.ModuleStrictGenerateCommon.HasTag(query)
}

func (m *ModuleGensrcs) GetTagsRegex(query *regexp.Regexp) []string {
	return m.ModuleStrictGenerateCommon.GetTagsRegex(query)
}

func (m *ModuleGensrcs) GetTags() []string {
	return m.ModuleStrictGenerateCommon.GetTags()
}

func gensrcsFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGensrcs{}

	module.ModuleStrictGenerateCommon.init(&config.Properties,
		StrictGenerateProps{}, GensrcsProps{}, EnableableProps{}, TagableProps{})

	return module, []interface{}{&module.ModuleStrictGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
