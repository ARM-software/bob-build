package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/google/blueprint"
)

type ModuleStaticLibrary struct {
	ModuleLibrary
}

var _ libraryInterface = (*ModuleStaticLibrary)(nil)             // impl check
var _ BackendConfiguration = (*ModuleStaticLibrary)(nil)         // impl check
var _ BackendConfigurationProvider = (*ModuleStaticLibrary)(nil) // impl check

func (m *ModuleStaticLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) && !isExternal(m) {
		getGenerator(ctx).staticActions(m, ctx)
	}
}

func (m *ModuleStaticLibrary) outputFileName() string {
	return m.outputName() + ".a"
}

func (m ModuleStaticLibrary) GetProperties() interface{} {
	return m.ModuleLibrary.Properties
}

func (m *ModuleStaticLibrary) OutFiles() (srcs file.Paths) {
	if !isExternal(m) {
		fp := file.NewPath(m.outputFileName(), string(m.getTarget()), file.TypeArchive|file.TypeInstallable)
		srcs = srcs.AppendIfUnique(fp)
	}
	return
}

func (m *ModuleStaticLibrary) OutFileTargets() []string {
	return []string{}
}

func (m *ModuleStaticLibrary) GetBackendConfiguration(ctx blueprint.ModuleContext) BackendConfiguration {
	return m
}

func (m *ModuleStaticLibrary) strip() bool {
	// Static library does not support stripping.
	return false
}

func staticLibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleStaticLibrary{}
	return module.LibraryFactory(config, module)
}
