package core

import (
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

type androidNinjaGenerator struct {
}

// aliasActions implements generatorBackend.
func (*androidNinjaGenerator) aliasActions(m *ModuleAlias, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// binaryActions implements generatorBackend.
func (*androidNinjaGenerator) binaryActions(m *ModuleBinary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// executableTestActions implements generatorBackend.
func (*androidNinjaGenerator) executableTestActions(m *ModuleTest, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// filegroupActions implements generatorBackend.
func (*androidNinjaGenerator) filegroupActions(m *ModuleFilegroup, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genBinaryActions implements generatorBackend.
func (*androidNinjaGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genSharedActions implements generatorBackend.
func (*androidNinjaGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genStaticActions implements generatorBackend.
func (*androidNinjaGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// generateSourceActions implements generatorBackend.
func (*androidNinjaGenerator) generateSourceActions(m *ModuleGenerateSource, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genruleActions implements generatorBackend.
func (*androidNinjaGenerator) genruleActions(m *ModuleGenrule, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// gensrcsActions implements generatorBackend.
func (*androidNinjaGenerator) gensrcsActions(m *ModuleGensrcs, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// kernelModuleActions implements generatorBackend.
func (*androidNinjaGenerator) kernelModuleActions(m *ModuleKernelObject, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// resourceActions implements generatorBackend.
func (*androidNinjaGenerator) resourceActions(m *ModuleResource, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// sharedActions implements generatorBackend.
func (*androidNinjaGenerator) sharedActions(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// staticActions implements generatorBackend.
func (*androidNinjaGenerator) staticActions(m *ModuleStaticLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// strictBinaryActions implements generatorBackend.
func (*androidNinjaGenerator) strictBinaryActions(m *ModuleStrictBinary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// strictLibraryActions implements generatorBackend.
func (*androidNinjaGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// transformSourceActions implements generatorBackend.
func (*androidNinjaGenerator) transformSourceActions(m *ModuleTransformSource, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// Compile time check for interface `androidNinjaGenerator` being compliant with generatorBackend
var _ generatorBackend = (*androidNinjaGenerator)(nil)
