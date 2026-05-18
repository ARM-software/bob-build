package core

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

type ImportCCBinaryProps struct {
	Src    string
	Target toolchain.TgtType
}

type ModuleImportCCBinary struct {
	module.ModuleBase
	Properties struct {
		SplittableProps
		ImportCCBinaryProps
	}
}

var _ splittable = (*ModuleImportCCBinary)(nil)
var _ file.Provider = (*ModuleImportCCBinary)(nil)

func importCCBinaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleImportCCBinary{}

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

func (g *linuxGenerator) importCCBinaryActions(m *ModuleImportCCBinary, ctx blueprint.ModuleContext) {
	addPhony(m, ctx, file.GetOutputs(m), false)
}

func (g *androidNinjaGenerator) importCCBinaryActions(m *ModuleImportCCBinary, ctx blueprint.ModuleContext) {

}

func (g *androidBpGenerator) importCCBinaryActions(m *ModuleImportCCBinary, ctx blueprint.ModuleContext) {

}

func (m *ModuleImportCCBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).importCCBinaryActions(m, ctx)
}

func (m *ModuleImportCCBinary) outputFileName() string {
	return m.Name()
}

func (m *ModuleImportCCBinary) shortName() string {
	return m.Name()
}

func (m *ModuleImportCCBinary) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.Src = filepath.Join(projectModuleDir(ctx), m.Properties.Src)
}

func (m *ModuleImportCCBinary) OutFiles() file.Paths {
	return file.Paths{
		file.NewPath(
			m.Properties.Src,
			file.FileNoNameSpace,
			file.TypeSrc|file.TypeInstallable,
		),
	}
}

func (m *ModuleImportCCBinary) OutFileTargets() (tgts []string) {
	return
}

// Support Splittable properties
func (m *ModuleImportCCBinary) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{m.Properties.Target}
}

func (m *ModuleImportCCBinary) setVariant(variant toolchain.TgtType) {
	// No need to actually track this, as a single target is always supported
}

func (m *ModuleImportCCBinary) disable() {
	// This should never actually be called, as we will always support one target
	panic("disable() called on ModuleImportCCBinary")
}

func (m *ModuleImportCCBinary) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *ModuleImportCCBinary) getTarget() toolchain.TgtType {
	return m.Properties.Target
}

// End Support Splittable properties
