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
	enableable
	file.Consumer
	Tagable
}

var _ strictLibraryInterface = (*ModuleStrictBinary)(nil)

func (m *ModuleStrictBinary) OutFiles() file.Paths {
	return file.Paths{
		file.NewPath(m.Name(), string(m.getTarget()), file.TypeBinary|file.TypeInstallable),
	}
}

func (m *ModuleStrictBinary) outputs() []string {
	return file.GetOutputs(m)
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
	module.Properties.Features.Init(&config.Properties, StrictLibraryProps{}, SplittableProps{}, InstallableProps{}, EnableableProps{}, IncludeProps{}, TagableProps{})
	module.Properties.Host.init(&config.Properties, StrictLibraryProps{}, InstallableProps{}, IncludeProps{}, TagableProps{})
	module.Properties.Target.init(&config.Properties, StrictLibraryProps{}, InstallableProps{}, IncludeProps{}, TagableProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
