package core

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

// TODO: Add Props one by one and test functionality of
// strip_include_prefix, cflags
type ImportCCLibraryProps struct {
	Src      string
	Target   toolchain.TgtType
	Linkopts []string
	Defines  []string
	Includes []string
}

type ModuleImportCCLibrary struct {
	module.ModuleBase
	Properties struct {
		SplittableProps
		ImportCCLibraryProps
	}
}

var _ splittable = (*ModuleImportCCLibrary)(nil)

type importCCLibraryInterface interface {
	splittable
	file.Provider
	flag.Provider
}

func (m *ModuleImportCCLibrary) isHeaderOnlyLib() bool {
	if m.Properties.Src == "" {
		return true
	}
	return false
}

func (m *ModuleImportCCLibrary) getLibFileType() file.Type {
	switch filepath.Ext(m.Properties.Src) {
	case ".so", ".dll", ".dylib":
		return file.TypeShared
	case ".a":
		return file.TypeArchive
	default:
		return file.TypeUnset
	}
}

func (m *ModuleImportCCLibrary) shortName() string {
	return m.Name()
}

func (m *ModuleImportCCLibrary) processPaths(ctx blueprint.BaseModuleContext) {
	prefix := projectModuleDir(ctx)
	m.Properties.Includes = utils.PrefixDirs(m.Properties.Includes, prefix)
	if !m.isHeaderOnlyLib() {
		m.Properties.Src = filepath.Join(prefix, m.Properties.Src)
	}
}

func (m *ModuleImportCCLibrary) OutFiles() (files file.Paths) {
	if !m.isHeaderOnlyLib() {
		files = append(files, file.NewPath(m.Properties.Src, file.FileNoNameSpace, file.TypeSrc|m.getLibFileType()))
	}

	return
}

func (m *ModuleImportCCLibrary) FlagsOut() (flags flag.Flags) {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Defines",
			Tag:          flag.TypeExported | flag.TypeTransitive,
			Factory:      flag.FromDefineOwned,
		},
		{
			PropertyName: "Linkopts",
			Tag:          flag.TypeTransitiveLinker,
			Factory:      flag.FromStringOwned,
		},
	}
	flags = append(flags, flag.ParseFromProperties(nil, lut, m.Properties)...)

	for _, dir := range m.Properties.Includes {
		fp := file.NewPath(dir, m.Name(), file.TypeHeader)
		flags = append(flags, flag.FromIncludePath(fp.BuildPath(), flag.TypeInclude|flag.TypeExported))
	}
	if !m.isHeaderOnlyLib() {
		if fp, ok := m.OutFiles().FindSingle(func(p file.Path) bool { return p.IsType(m.getLibFileType()) }); ok {
			flags = append(flags, flag.FromString(fp.BuildPath(), flag.TypeLinkLibrary|flag.TypeExported))
		}
	}
	return
}

func (m *ModuleImportCCLibrary) exportSharedLibs() []string { return []string{} }

func importCCLibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleImportCCLibrary{}

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

func (g *linuxGenerator) importCCLibraryActions(m *ModuleImportCCLibrary, ctx blueprint.ModuleContext) {
	addPhony(m, ctx, nil, false)
}

func (g *androidNinjaGenerator) importCCLibraryActions(m *ModuleImportCCLibrary, ctx blueprint.ModuleContext) {

}

// TODO: Does android need to generate anything? A "promise" that'll exist?
func (g *androidBpGenerator) importCCLibraryActions(m *ModuleImportCCLibrary, ctx blueprint.ModuleContext) {

}

func (m *ModuleImportCCLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).importCCLibraryActions(m, ctx)
}

// Support Splittable properties
func (m *ModuleImportCCLibrary) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{m.Properties.Target}
}

func (m *ModuleImportCCLibrary) setVariant(variant toolchain.TgtType) {
	// No need to actually track this, as a single target is always supported
}

func (m *ModuleImportCCLibrary) disable() {
	// This should never actually be called, as we will always support one target
	panic("disable() called on ModuleImportCCLibrary")
}

func (m *ModuleImportCCLibrary) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *ModuleImportCCLibrary) getTarget() toolchain.TgtType {
	return m.Properties.Target
}

// End Support Splittable properties
