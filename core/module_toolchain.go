package core

import (
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

type ModuleToolchainProps struct {
	// Flags that will be used for C and C++ compiles.
	Cflags []string

	// Flags that will be used for C compiles.
	Conlyflags []string

	// Flags that will be used for C++ compiles.
	Cppflags []string

	// Flags that will be used for .S compiles.
	Asflags []string

	// Flags that will be used for all link steps.
	Ldflags []string
}

type ToolchainFlagsProps struct {
	// `ModuleToolchain` module.
	Toolchain *string
}

// Strict targets will not support defaults by design.
//
// With this in mind, we will need a way to propagate
// common toolchain flags to targets (optimization etc).
type ModuleToolchain struct {
	module.ModuleBase

	SplittableProps

	Properties struct {
		ModuleToolchainProps

		Target     TargetSpecific
		Host       TargetSpecific
		TargetType toolchain.TgtType `blueprint:"mutated"`

		Features
	}
}

type ModuleToolchainInterface interface {
	Featurable
	targetSpecificLibrary
	flag.Provider
}

var _ ModuleToolchainInterface = (*ModuleToolchain)(nil)

func (m *ModuleToolchain) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.ModuleToolchainProps,
	}
}

func (m *ModuleToolchain) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleToolchain) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// `ModuleToolchain` does not generate any actions.
	// It only provides flags to be consumed by other modules.
}

func (m *ModuleToolchain) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{toolchain.TgtTypeHost, toolchain.TgtTypeTarget}
}

func (m *ModuleToolchain) disable() {
	// always enabled
}

func (m *ModuleToolchain) setVariant(tgt toolchain.TgtType) {
	m.Properties.TargetType = tgt
}

func (m *ModuleToolchain) getTarget() toolchain.TgtType {
	return m.Properties.TargetType
}

func (m *ModuleToolchain) getSplittableProps() *SplittableProps {
	return &m.SplittableProps
}

func (m *ModuleToolchain) getTargetSpecific(tgt toolchain.TgtType) *TargetSpecific {
	if tgt == toolchain.TgtTypeHost {
		return &m.Properties.Host
	} else if tgt == toolchain.TgtTypeTarget {
		return &m.Properties.Target
	}

	return nil
}

// Get the set of the module main properties for
// that target specific properties would be applied to
func (m *ModuleToolchain) targetableProperties() []interface{} {
	return []interface{}{
		&m.Properties.ModuleToolchainProps,
	}
}

func (m *ModuleToolchain) FlagsOut() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Cflags",
			Tag:          flag.TypeCC | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Conlyflags",
			Tag:          flag.TypeC | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Cppflags",
			Tag:          flag.TypeCpp | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Asflags",
			Tag:          flag.TypeAsm | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Ldflags",
			Tag:          flag.TypeLinker | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
	}

	return flag.ParseFromProperties(nil, lut, m.Properties)
}

func ModuleToolchainFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleToolchain{}

	module.Properties.Features.Init(&config.Properties, ModuleToolchainProps{})
	module.Properties.Host.init(&config.Properties, ModuleToolchainProps{})
	module.Properties.Target.init(&config.Properties, ModuleToolchainProps{})

	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
