package core

import "github.com/google/blueprint"

type ModuleStrictBinary struct {
	ModuleStrictLibrary
}

type strictBinaryInterface interface {
	splittable
	FileConsumer
}

func (m *ModuleStrictBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).strictBinaryActions(m, ctx)
	}
}
