package core

import (
	"github.com/ARM-software/bob-build/core/module"
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

// Strict targets will not support defaults by design.
//
// With this in mind, we will need a way to propagate
// common toolchain flags to targets (optimization etc).
type ModuleToolchain struct {
	module.ModuleBase

	SplittableProps

	Properties struct {
		ModuleToolchainProps

		Target TargetSpecific
		Host   TargetSpecific

		Features
	}
}

type ModuleToolchainInterface interface {
	Featurable
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

func ModuleToolchainFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleToolchain{}

	module.Properties.Features.Init(&config.Properties, ModuleToolchainProps{})
	module.Properties.Host.init(&config.Properties, ModuleToolchainProps{})
	module.Properties.Target.init(&config.Properties, ModuleToolchainProps{})

	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
