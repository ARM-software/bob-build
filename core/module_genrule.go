package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

/*
	We are swapping from `bob_generate_source` to `bob_genrule`

`bob_genrule` is made to be a stricter version that is compatible with Android.
For easiest compatibility, we are using Androids format for `genrule`.
Some properties in the struct may not be useful, but it is better to expose as many
features as possible rather than too few. Some are commented out as they would take special
implementation for features we do not already have in place.

*/

type GenruleProps struct {
	Out         []string
	ResolvedOut file.Paths `blueprint:"mutated"`
}

type ModuleGenrule struct {
	ModuleStrictGenerateCommon
	Properties struct {
		GenruleProps
	}
}

type ModuleGenruleInterface interface {
	file.Consumer
	file.Resolver
	pathProcessor
}

var _ ModuleGenruleInterface = (*ModuleGenrule)(nil) // impl check

func checkGenruleFieldsMutator(ctx blueprint.BottomUpMutatorContext) {
	m := ctx.Module()
	if b, ok := m.(*ModuleGenrule); ok {
		props := b.ModuleStrictGenerateCommon.Properties
		if len(props.Export_include_dirs) != 0 {
			utils.Die("`export_include_dirs` may lead to unexpected results on AOSP for `bob_genrule`, please use `bob_gensrc` rule type instead. In module %s", m.Name())
		}
	}
}

func (m *ModuleGenrule) implicitOutputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsType(file.TypeImplicit) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleGenrule) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsNotType(file.TypeDep) && f.IsNotType(file.TypeImplicit) },
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

	if proptools.Bool(m.ModuleStrictGenerateCommon.Properties.Depfile) {
		files = append(files, file.NewPath(utils.FlattenPath(m.Name())+".d", m.Name(), file.TypeDep|file.TypeGenerated))
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
	gc := m.getStrictGenerateCommon()
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

func (m *ModuleGenrule) getStrictGenerateCommon() *ModuleStrictGenerateCommon {
	return &m.ModuleStrictGenerateCommon
}

func generateRuleAndroidFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleGenrule{}

	module.ModuleStrictGenerateCommon.init(&config.Properties,
		StrictGenerateProps{}, GenruleProps{}, EnableableProps{})

	return module, []interface{}{&module.ModuleStrictGenerateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
