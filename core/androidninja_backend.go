package core

import (
	"github.com/google/blueprint"
)

type androidNinjaGenerator struct {
}

// aliasActions implements generatorBackend.
func (*androidNinjaGenerator) aliasActions(*ModuleAlias, blueprint.ModuleContext) {
	panic("unimplemented")
}

// binaryActions implements generatorBackend.
func (*androidNinjaGenerator) binaryActions(*ModuleBinary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// executableTestActions implements generatorBackend.
func (*androidNinjaGenerator) executableTestActions(*ModuleTest, blueprint.ModuleContext) {
	panic("unimplemented")
}

// filegroupActions implements generatorBackend.
func (*androidNinjaGenerator) filegroupActions(*ModuleFilegroup, blueprint.ModuleContext) {
	panic("unimplemented")
}

// genBinaryActions implements generatorBackend.
func (*androidNinjaGenerator) genBinaryActions(*generateBinary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// genSharedActions implements generatorBackend.
func (*androidNinjaGenerator) genSharedActions(*generateSharedLibrary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// genStaticActions implements generatorBackend.
func (*androidNinjaGenerator) genStaticActions(*generateStaticLibrary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// generateSourceActions implements generatorBackend.
func (*androidNinjaGenerator) generateSourceActions(*ModuleGenerateSource, blueprint.ModuleContext) {
	panic("unimplemented")
}

// genruleActions implements generatorBackend.
func (*androidNinjaGenerator) genruleActions(*ModuleGenrule, blueprint.ModuleContext) {
	panic("unimplemented")
}

// gensrcsActions implements generatorBackend.
func (*androidNinjaGenerator) gensrcsActions(*ModuleGensrcs, blueprint.ModuleContext) {
	panic("unimplemented")
}

// kernelModuleActions implements generatorBackend.
func (*androidNinjaGenerator) kernelModuleActions(*ModuleKernelObject, blueprint.ModuleContext) {
	panic("unimplemented")
}

// resourceActions implements generatorBackend.
func (*androidNinjaGenerator) resourceActions(*ModuleResource, blueprint.ModuleContext) {
	panic("unimplemented")
}

// sharedActions implements generatorBackend.
func (*androidNinjaGenerator) sharedActions(*ModuleSharedLibrary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// staticActions implements generatorBackend.
func (*androidNinjaGenerator) staticActions(*ModuleStaticLibrary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// strictBinaryActions implements generatorBackend.
func (*androidNinjaGenerator) strictBinaryActions(*ModuleStrictBinary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// strictLibraryActions implements generatorBackend.
func (*androidNinjaGenerator) strictLibraryActions(*ModuleStrictLibrary, blueprint.ModuleContext) {
	panic("unimplemented")
}

// transformSourceActions implements generatorBackend.
func (*androidNinjaGenerator) transformSourceActions(*ModuleTransformSource, blueprint.ModuleContext) {
	panic("unimplemented")
}

// Compile time check for interface `androidNinjaGenerator` being compliant with generatorBackend
var _ generatorBackend = (*androidNinjaGenerator)(nil)
