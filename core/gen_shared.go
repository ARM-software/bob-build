package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/google/blueprint"
)

type generateSharedLibrary struct {
	generateLibrary
	fileNameExtension string
}

// Verify that the following interfaces are implemented
var _ file.Provider = (*generateSharedLibrary)(nil)
var _ generateLibraryInterface = (*generateSharedLibrary)(nil)
var _ singleOutputModule = (*generateSharedLibrary)(nil)
var _ sharedLibProducer = (*generateSharedLibrary)(nil)
var _ splittable = (*generateSharedLibrary)(nil)
var _ blueprint.Module = (*generateSharedLibrary)(nil)

func (m *generateSharedLibrary) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	return generateLibraryInouts(m, ctx, g, m.Properties.Headers)
}

func (m *generateSharedLibrary) OutFiles() (files file.Paths) {
	gc, _ := getGenerateCommon(m)
	files = append(files, gc.OutFiles()...)

	files = append(files, file.NewPath(m.outputFileName(), m.Name(), file.TypeGenerated|file.TypeInstallable))

	toc := file.NewPath(m.getTocName(), string(m.getTarget()), file.TypeImplicit)
	files = append(files, toc)

	for _, h := range m.Properties.Headers {
		fp := file.NewPath(h, m.Name(), file.TypeGenerated|file.TypeHeader|file.TypeImplicit)
		files = append(files, fp)
	}

	return
}

func (m *generateSharedLibrary) OutFileTargets() []string { return []string{} }

func (m *generateSharedLibrary) FlagsOut() (flags flag.Flags) {
	gc, _ := getGenerateCommon(m)
	for _, str := range gc.Properties.Export_gen_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

//// Support generateLibraryInterface

func (m *generateSharedLibrary) libExtension() string {
	return m.fileNameExtension
}

//// Support blueprint.Module

func (m *generateSharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).genSharedActions(m, ctx)
	}
}

//// Support singleOutputModule

func (m *generateSharedLibrary) outputFileName() string {
	return m.altName() + m.libExtension()
}

//// Support sharedLibProducer

func (m *generateSharedLibrary) getTocName() string {
	return m.outputFileName() + tocExt
}

func (m generateSharedLibrary) GetProperties() interface{} {
	return m.generateLibrary.Properties
}

//// Factory functions

func genSharedLibFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateSharedLibrary{}
	module.ModuleGenerateCommon.init(&config.Properties, GenerateProps{},
		GenerateLibraryProps{})

	if config.Properties.GetBool("osx") {
		module.fileNameExtension = ".dylib"
	} else {
		module.fileNameExtension = ".so"
	}
	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.ModuleGenerateCommon.Properties,
		&module.Properties,
	}
}
