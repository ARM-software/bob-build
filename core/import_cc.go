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
type ImportCCProps struct {
	Src      string
	Target   toolchain.TgtType
	Linkopts []string
	Defines  []string
	Includes []string
}

type ModuleImportCC struct {
	module.ModuleBase
	Properties struct {
		SplittableProps
		ImportCCProps
	}
}

var _ splittable = (*ModuleImportCC)(nil)

type importCCInterface interface {
	splittable
	file.Provider
	flag.Provider
}

func (m *ModuleImportCC) isHeaderOnlyLib() bool {
	if m.Properties.Src == "" {
		return true
	}
	return false
}

func (m *ModuleImportCC) getLibFileType() file.Type {
	switch filepath.Ext(m.Properties.Src) {
	case ".so", ".dll", ".dylib":
		return file.TypeShared
	case ".a":
		return file.TypeArchive
	default:
		return file.TypeUnset
	}
}

func (m *ModuleImportCC) shortName() string {
	return m.Name()
}

func (m *ModuleImportCC) processPaths(ctx blueprint.BaseModuleContext) {
	prefix := projectModuleDir(ctx)
	m.Properties.Includes = utils.PrefixDirs(m.Properties.Includes, prefix)
	if !m.isHeaderOnlyLib() {
		m.Properties.Src = filepath.Join(prefix, m.Properties.Src)
	}
}

func (m *ModuleImportCC) OutFiles() (files file.Paths) {
	if !m.isHeaderOnlyLib() {
		files = append(files, file.NewPath(m.Properties.Src, file.FileNoNameSpace, file.TypeSrc|m.getLibFileType()))
	}

	return
}

func (m *ModuleImportCC) FlagsOut() (flags flag.Flags) {
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

func (m *ModuleImportCC) exportSharedLibs() []string { return []string{} }

func importCCFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleImportCC{}

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

func (g *linuxGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {
	addPhony(m, ctx, nil, false)
}

func (g *androidNinjaGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {

}

// TODO: Does android need to generate anything? A "promise" that'll exist?
func (g *androidBpGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {

}

func (m *ModuleImportCC) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).importCCActions(m, ctx)
}

// Support Splittable properties
func (m *ModuleImportCC) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{m.Properties.Target}
}

func (m *ModuleImportCC) setVariant(variant toolchain.TgtType) {
	// No need to actually track this, as a single target is always supported
}

func (m *ModuleImportCC) disable() {
	// This should never actually be called, as we will always support one target
	panic("disable() called on ModuleImportCC")
}

func (m *ModuleImportCC) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *ModuleImportCC) getTarget() toolchain.TgtType {
	return m.Properties.Target
}

// End Support Splittable properties
