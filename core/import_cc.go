package core

import (
	"path/filepath"
	"strings"

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
	if strings.HasSuffix(m.Name(), ".so") {
		return file.TypeShared
	}
	if strings.HasSuffix(m.Name(), ".a") {
		return file.TypeArchive
	}
	return file.TypeUnset
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
		lib := m.Properties.Src
		src := file.NewPath(lib, m.Name(), m.getLibFileType())
		fp := file.NewLink(lib, m.Name(), &src, m.getLibFileType()|file.TypeGenerated)
		files = append(files, fp)
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
		flags = append(flags, flag.FromString(pathToLibFlag(m.Properties.Src), flag.TypeLinker))
	}
	return
}

func importCCFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleImportCC{}

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

func (g *linuxGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {
	installDeps := []string{}
	m.OutFiles().ForEachIf(
		func(fp file.Path) bool { return fp.IsSymLink() },
		func(fp file.Path) bool {
			if relative, err := filepath.Rel(fp.RelBuildPath(), fp.ExpandLink().RelBuildPath()); err == nil {
				ctx.Build(pctx,
					blueprint.BuildParams{
						Rule:     symlinkRule,
						Inputs:   []string{fp.ExpandLink().BuildPath()},
						Outputs:  []string{fp.BuildPath()},
						Args:     map[string]string{"target": relative},
						Optional: true,
					})
				installDeps = append(installDeps, fp.BuildPath())
				return true
			}

			return false
		})

	addPhony(m, ctx, installDeps, false) // Always add the symlinks
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
