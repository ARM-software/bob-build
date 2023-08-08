package core

import (
	"github.com/ARM-software/bob-build/core/module"
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
