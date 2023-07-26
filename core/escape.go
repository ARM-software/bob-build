package core

import (
	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/internal/escape"
)

type propertyEscapeInterface interface {
	getEscapeProperties() []*[]string
}

func escapeMutator(ctx blueprint.TopDownMutatorContext) {
	// This mutator is not registered on the androidbp backend, as it
	// doesn't need escaping
	module := ctx.Module()

	if _, ok := module.(*ModuleDefaults); ok {
		// No need to apply to defaults
		return
	}

	if e, ok := module.(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, skip execution
			return
		}
	}

	// Escape libraries as well as generator modules
	if m, ok := module.(propertyEscapeInterface); ok {
		escapeProps := m.getEscapeProperties()

		for _, prop := range escapeProps {
			// If the flags contain template sequences, we avoid escaping those
			*prop = escape.EscapeTemplatedStringList(*prop, backend.Get().EscapeFlag)
		}
	}
}
