package core

import (
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"

	"github.com/google/blueprint"
)

type ExternalLibProps struct {
	Export_cflags  []string
	Export_ldflags []string
	Ldlibs         []string

	TargetType toolchain.TgtType `blueprint:"mutated"`
}

type ModuleExternalLibrary struct {
	module.ModuleBase
	Properties struct {
		ExternalLibProps
		Features
	}
}

func (m *ModuleExternalLibrary) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.ExternalLibProps}
}

func (m *ModuleExternalLibrary) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleExternalLibrary) outputName() string   { return m.Name() }
func (m *ModuleExternalLibrary) altName() string      { return m.outputName() }
func (m *ModuleExternalLibrary) altShortName() string { return m.altName() }
func (m *ModuleExternalLibrary) shortName() string    { return m.Name() }

// External libraries have no outputs - they are already built.
func (m *ModuleExternalLibrary) outputs() []string         { return []string{} }
func (m *ModuleExternalLibrary) implicitOutputs() []string { return []string{} }

// Implement the splittable interface so "normal" libraries can depend on external ones.
func (m *ModuleExternalLibrary) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{toolchain.TgtTypeHost, toolchain.TgtTypeTarget}
}
func (m *ModuleExternalLibrary) disable()                             {}
func (m *ModuleExternalLibrary) setVariant(tgt toolchain.TgtType)     { m.Properties.TargetType = tgt }
func (m *ModuleExternalLibrary) getTarget() toolchain.TgtType         { return m.Properties.TargetType }
func (m *ModuleExternalLibrary) getSplittableProps() *SplittableProps { return &SplittableProps{} }

// Implement the propertyExporter interface so that external libraries can pass
// on properties e.g. from pkg-config

func (m *ModuleExternalLibrary) exportCflags() []string                 { return m.Properties.Export_cflags }
func (m *ModuleExternalLibrary) exportIncludeDirs() []string            { return []string{} }
func (m *ModuleExternalLibrary) exportLocalIncludeDirs() []string       { return []string{} }
func (m *ModuleExternalLibrary) exportLdflags() []string                { return m.Properties.Export_ldflags }
func (m *ModuleExternalLibrary) exportLdlibs() []string                 { return m.Properties.Ldlibs }
func (m *ModuleExternalLibrary) exportSharedLibs() []string             { return []string{} }
func (m *ModuleExternalLibrary) exportSystemIncludeDirs() []string      { return []string{} }
func (m *ModuleExternalLibrary) exportLocalSystemIncludeDirs() []string { return []string{} }

func (m *ModuleExternalLibrary) FlagsIn() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Export_cflags",
			Tag:          flag.TypeCC,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Export_ldflags",
			Tag:          flag.TypeLinker,
			Factory:      flag.FromStringOwned,
		},
	}

	return flag.ParseFromProperties(nil, lut, m.Properties)
}

func (m *ModuleExternalLibrary) FlagsInTransitive(ctx blueprint.BaseModuleContext) (ret flag.Flags) {
	m.FlagsIn().ForEach(
		func(f flag.Flag) {
			ret = append(ret, f)
		})

	flag.ReferenceFlagsInTransitive(ctx).ForEach(
		func(f flag.Flag) {
			ret = append(ret, f)
		})

	return
}

func (m *ModuleExternalLibrary) FlagsOut() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Export_cflags",
			Tag:          flag.TypeCC | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Export_ldflags",
			Tag:          flag.TypeLinker | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
	}

	return flag.ParseFromProperties(nil, lut, m.Properties)
}

var _ propertyExporter = (*ModuleExternalLibrary)(nil)
var _ splittable = (*ModuleExternalLibrary)(nil)

// External libraries have no actions - they are already built.
func (m *ModuleExternalLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {}

func (m ModuleExternalLibrary) GetProperties() interface{} {
	return m.Properties
}

func externalLibFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleExternalLibrary{}
	module.Properties.Features.Init(&config.Properties, ExternalLibProps{})
	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
