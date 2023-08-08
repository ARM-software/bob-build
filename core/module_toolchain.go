package core

import (
	"github.com/ARM-software/bob-build/core/module"
)

// Strict targets will not support defaults by design.
//
// With this in mind, we will need a way to propagate
// common toolchain flags to targets (optimization etc).
type ModuleToolchain struct {
	module.ModuleBase
}
