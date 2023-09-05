package core

import "github.com/google/blueprint"

type ModuleTest struct {
	ModuleStrictBinary
}

func (m *ModuleTest) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).executableTestActions(m, ctx)
	}
}

func executableTestFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	t := true

	module := &ModuleTest{}
	module.Properties.Linkstatic = &t // always true for executables
	module.Properties.Features.Init(&config.Properties, StrictLibraryProps{}, SplittableProps{}, InstallableProps{}, EnableableProps{}, IncludeProps{})
	module.Properties.Host.init(&config.Properties, StrictLibraryProps{}, InstallableProps{}, IncludeProps{})
	module.Properties.Target.init(&config.Properties, StrictLibraryProps{}, InstallableProps{}, IncludeProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
