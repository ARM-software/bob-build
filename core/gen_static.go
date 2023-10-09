package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/google/blueprint"
)

type generateStaticLibrary struct {
	generateLibrary
}

// Verify that the following interfaces are implemented
var _ file.Provider = (*generateStaticLibrary)(nil)
var _ generateLibraryInterface = (*generateStaticLibrary)(nil)
var _ singleOutputModule = (*generateStaticLibrary)(nil)
var _ splittable = (*generateStaticLibrary)(nil)
var _ blueprint.Module = (*generateStaticLibrary)(nil)

func (m *generateStaticLibrary) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	return generateLibraryInouts(m, ctx, g, m.Properties.Headers)
}

func (m *generateStaticLibrary) implicitOutputs() []string {
	return m.OutFiles().ToStringSliceIf(
		// TODO: ideally we should just check for `TypeImplicit` here,
		// but currently set up to mirror existing behaviour
		func(f file.Path) bool { return f.IsNotType(file.TypeArchive) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *generateStaticLibrary) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		// TODO: ideally we should just check for `TypeImplicit` here,
		// but currently set up to mirror existing behaviour
		func(f file.Path) bool { return f.IsType(file.TypeArchive) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *generateStaticLibrary) OutFiles() (files file.Paths) {
	gc, _ := getGenerateCommon(m)
	files = append(files, gc.OutFiles()...)

	files = append(files, file.NewPath(m.outputFileName(), m.Name(), file.TypeGenerated|file.TypeInstallable))

	for _, h := range m.Properties.Headers {
		fp := file.NewPath(h, m.Name(), file.TypeGenerated|file.TypeHeader)
		files = append(files, fp)
	}

	return
}

func (m *generateStaticLibrary) FlagsOut() (flags flag.Flags) {
	gc, _ := getGenerateCommon(m)
	for _, str := range gc.Properties.Export_gen_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

//// Support generateLibraryInterface

func (m *generateStaticLibrary) libExtension() string {
	return ".a"
}

//// Support blueprint.Module

func (m *generateStaticLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getGenerator(ctx)
		g.genStaticActions(m, ctx)
	}
}

//// Support singleOutputModule

func (m *generateStaticLibrary) outputFileName() string {
	return m.altName() + m.libExtension()
}

func (m generateStaticLibrary) GetProperties() interface{} {
	return m.generateLibrary.Properties
}

//// Factory functions

func genStaticLibFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateStaticLibrary{}
	module.ModuleGenerateCommon.init(&config.Properties, GenerateProps{},
		GenerateLibraryProps{})

	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.ModuleGenerateCommon.Properties,
		&module.Properties,
	}
}
