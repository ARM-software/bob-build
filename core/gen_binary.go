package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/google/blueprint"
)

type generateBinary struct {
	generateLibrary
}

// Verify that the following interfaces are implemented
var _ FileProvider = (*generateBinary)(nil)
var _ generateLibraryInterface = (*generateBinary)(nil)
var _ singleOutputModule = (*generateBinary)(nil)
var _ splittable = (*generateBinary)(nil)
var _ blueprint.Module = (*generateBinary)(nil)

func (m *generateBinary) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	return generateLibraryInouts(m, ctx, g, m.Properties.Headers)
}

func (m *generateBinary) implicitOutputs() []string {
	return []string{}
}

func (m *generateBinary) outputs() []string {
	return m.OutFiles().ToStringSlice(func(f file.Path) string { return f.BuildPath() })
}

func (m *generateBinary) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	return m.outputs()
}

func (m *generateBinary) OutFiles() (files file.Paths) {
	return file.Paths{
		file.NewPath(
			m.outputName(),
			m.Name(),
			file.TypeBinary|file.TypeExecutable|file.TypeGenerated,
		),
	}
}

//// Support generateLibraryInterface

func (m *generateBinary) libExtension() string {
	return ""
}

//// Support singleOutputModule

func (m *generateBinary) outputFileName() string {
	return m.altName() + m.libExtension()
}

//// Support blueprint.Module

func (m *generateBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).genBinaryActions(m, ctx)
	}
}

func (m generateBinary) GetProperties() interface{} {
	return m.generateLibrary.Properties
}

//// Factory functions

func genBinaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateBinary{}
	module.ModuleGenerateCommon.init(&config.Properties, GenerateProps{})

	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.ModuleGenerateCommon.Properties,
		&module.Properties,
	}
}
