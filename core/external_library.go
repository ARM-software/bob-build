package core

import (
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
)

type ExternalLibProps struct {
	Export_cflags  []string
	Export_ldflags []string
	Ldlibs         []string
	Target         TargetSpecific
	Host           TargetSpecific
	SplittableProps
	TargetType toolchain.TgtType `blueprint:"mutated"`
}

type ModuleExternalLibrary struct {
	module.ModuleBase
	Properties struct {
		ExternalLibProps
		TagableProps
		Features
		EnableableProps
	}
}

func (m *ModuleExternalLibrary) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.ExternalLibProps,
		&m.Properties.TagableProps,
		&m.Properties.SplittableProps,
	}
}

func (m *ModuleExternalLibrary) targetableProperties() []interface{} {
	return []interface{}{
		&m.Properties.ExternalLibProps,
	}
}

func (m *ModuleExternalLibrary) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleExternalLibrary) outputName() string   { return m.Name() }
func (m *ModuleExternalLibrary) altName() string      { return m.outputName() }
func (m *ModuleExternalLibrary) altShortName() string { return m.altName() }
func (m *ModuleExternalLibrary) shortName() string    { return m.Name() }

func (b *ExternalLibProps) isHostSupported() bool {
	if b.Host_supported == nil {
		return false
	}
	return *b.Host_supported
}

func (b *ExternalLibProps) isTargetSupported() bool {
	if b.Target_supported == nil {
		return true
	}
	return *b.Target_supported
}

func (m *ModuleExternalLibrary) supportedVariants() (tgts []toolchain.TgtType) {
	if m.Properties.isHostSupported() {
		tgts = append(tgts, toolchain.TgtTypeHost)
	}
	if m.Properties.isTargetSupported() {
		tgts = append(tgts, toolchain.TgtTypeTarget)
	}
	return
}
func (m *ModuleExternalLibrary) disable() {
	f := false
	m.Properties.Enabled = &f
}

func (m *ModuleExternalLibrary) setVariant(tgt toolchain.TgtType) { m.Properties.TargetType = tgt }
func (m *ModuleExternalLibrary) getTarget() toolchain.TgtType     { return m.Properties.TargetType }
func (m *ModuleExternalLibrary) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (l ExternalLibProps) getTargetSpecific(tgt toolchain.TgtType) *TargetSpecific {
	if tgt == toolchain.TgtTypeHost {
		return &l.Host
	} else if tgt == toolchain.TgtTypeTarget {
		return &l.Target
	} else {
		utils.Die("Unsupported target type: %s", tgt)
	}
	return nil
}

func (m *ModuleExternalLibrary) getTargetSpecific(tgt toolchain.TgtType) *TargetSpecific {
	return m.Properties.getTargetSpecific(tgt)
}

// Implement the SharedLibraryExporter interface so that external libraries can pass
// on properties e.g. from pkg-config

func (m *ModuleExternalLibrary) exportSharedLibs() []string { return []string{} }

func (m *ModuleExternalLibrary) FlagsIn() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Ldlibs",
			Tag:          flag.TypeLinkLibrary,
			Factory:      flag.FromStringOwned,
		},
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
			PropertyName: "Ldlibs",
			Tag:          flag.TypeLinkLibrary | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
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

var _ SharedLibraryExporter = (*ModuleExternalLibrary)(nil)
var _ splittable = (*ModuleExternalLibrary)(nil)
var _ targetSpecificLibrary = (*ModuleExternalLibrary)(nil) // impl check

// External libraries have no actions - they are already built.
func (m *ModuleExternalLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {}

func (m ModuleExternalLibrary) GetProperties() interface{} {
	return m.Properties
}

func externalLibFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleExternalLibrary{}
	module.Properties.Features.Init(&config.Properties, ExternalLibProps{}, SplittableProps{})

	module.Properties.Host.init(&config.Properties, ExternalLibProps{})
	module.Properties.Target.init(&config.Properties, ExternalLibProps{})
	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

var _ externableLibrary = (*ModuleExternalLibrary)(nil) // impl check
func (m *ModuleExternalLibrary) isExternal() bool {
	return true
}
