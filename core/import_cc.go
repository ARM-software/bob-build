package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/google/blueprint"
)

// TODO: Add Props one by one and test functionality of
// headers, defines, `src` aka library, strip_include_prefix
type ImportCCProps struct {
}

type ModuleImportCC struct {
	module.ModuleBase
	Properties struct {
		SplittableProps
		ImportCCProps
	}
}

type importCCInterface interface {
	splittable
	file.Provider
}

func importCCFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleImportCC{}

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

func (g *linuxGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {

}

func (g *androidNinjaGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {

}

// TODO: Does android need to generate anything? A "promise" that'll exist?
func (g *androidBpGenerator) importCCActions(m *ModuleImportCC, ctx blueprint.ModuleContext) {

}

func (m *ModuleImportCC) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).importCCActions(m, ctx)
}
