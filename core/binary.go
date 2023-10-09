package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/google/blueprint"
)

type ModuleBinary struct {
	ModuleLibrary
}

// binary supports:
type binaryInterface interface {
	stripable
	linkableModule
	file.Provider // A binary can provide itself as a source
	StripCapable
}

var _ binaryInterface = (*ModuleBinary)(nil)  // impl check
var _ libraryInterface = (*ModuleBinary)(nil) // impl check

func (m *ModuleBinary) implicitOutputs() []string {
	return file.GetImplicitOutputs(m)
}

func (m *ModuleBinary) outputs() []string {
	return m.OutFiles().ToStringSlice(func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleBinary) OutFiles() (srcs file.Paths) {
	return file.Paths{file.NewPath(m.outputName(), string(m.getTarget()), file.TypeBinary|file.TypeExecutable|file.TypeInstallable)}
}

func (m *ModuleBinary) OutFileTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleBinary) strip() bool {
	return m.Properties.Strip != nil && *m.Properties.Strip
}

func (m *ModuleBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).binaryActions(m, ctx)
	}
}

func (m *ModuleBinary) outputFileName() string {
	return m.outputName()
}

func (m ModuleBinary) GetProperties() interface{} {
	return m.ModuleLibrary.Properties
}

func (m *ModuleBinary) GetStripable(ctx blueprint.ModuleContext) stripable {
	return m
}

func binaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleBinary{}
	return module.LibraryFactory(config, module)
}
