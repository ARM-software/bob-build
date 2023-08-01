package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/google/blueprint"
)

type ModuleStrictBinary struct {
	ModuleStrictLibrary
}

type strictBinaryInterface interface {
	splittable
	FileConsumer
}

func (m *ModuleStrictBinary) OutFiles() file.Paths {
	return file.Paths{
		file.NewPath(m.Name(), string(m.getTarget()), file.TypeBinary),
	}
}

func (m *ModuleStrictBinary) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return f.IsType(file.TypeBinary) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleStrictBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).strictBinaryActions(m, ctx)
	}
}

func StrictBinaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	t := true

	module := &ModuleStrictBinary{}
	module.Properties.Linkstatic = &t // always true for executables
	module.Properties.Features.Init(&config.Properties, StrictLibraryProps{}, SplittableProps{})
	module.Properties.Host.init(&config.Properties, StrictLibraryProps{})
	module.Properties.Target.init(&config.Properties, StrictLibraryProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
