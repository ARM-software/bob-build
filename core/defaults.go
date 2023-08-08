package core

import (
	"sync"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"

	"github.com/google/blueprint"
)

type ModuleDefaults struct {
	module.ModuleBase

	Properties struct {
		Features
		Build
		KernelProps
		// The list of default properties that should prepended to all configuration
		Defaults []string
	}
}

func (m *ModuleDefaults) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{toolchain.TgtTypeHost, toolchain.TgtTypeTarget}
}

func (m *ModuleDefaults) disable() {
	panic("disable() called on Default")
}

func (m *ModuleDefaults) setVariant(variant toolchain.TgtType) {
	m.Properties.TargetType = variant
}

func (m *ModuleDefaults) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *ModuleDefaults) defaults() []string {
	return m.Properties.Defaults
}

func (m *ModuleDefaults) build() *Build {
	return &m.Properties.Build
}

func (m *ModuleDefaults) defaultableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.KernelProps,
	}
}

func (m *ModuleDefaults) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.KernelProps,
	}
}

func (m *ModuleDefaults) targetableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.KernelProps,
	}
}

func (m *ModuleDefaults) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleDefaults) getTarget() toolchain.TgtType {
	return m.Properties.TargetType
}

func (m *ModuleDefaults) getTargetSpecific(variant toolchain.TgtType) *TargetSpecific {
	return m.Properties.getTargetSpecific(variant)
}

func (m *ModuleDefaults) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.Build.processPaths(ctx)
	m.Properties.KernelProps.processPaths(ctx)
}

func (m *ModuleDefaults) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

func (m *ModuleDefaults) getEscapeProperties() []*[]string {
	return []*[]string{
		&m.Properties.Asflags,
		&m.Properties.Cflags,
		&m.Properties.Conlyflags,
		&m.Properties.Cxxflags,
		&m.Properties.Ldflags}
}

func (m *ModuleDefaults) getLegacySourceProperties() *LegacySourceProps {
	return &m.Properties.LegacySourceProps
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `BuildProps` this is:
//   - Ldflags
//   - Cflags
//   - Conlyflags
//   - Cxxflags
func (m *ModuleDefaults) getMatchSourcePropNames() []string {
	return []string{"Ldflags", "Cflags", "Conlyflags", "Cxxflags"}
}

func (m ModuleDefaults) GetProperties() interface{} {
	return m.Properties
}

func defaultsFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleDefaults{}

	module.Properties.Features.Init(&config.Properties, CommonProps{}, BuildProps{}, KernelProps{}, SplittableProps{})
	module.Properties.Host.init(&config.Properties, CommonProps{}, BuildProps{}, KernelProps{})
	module.Properties.Target.init(&config.Properties, CommonProps{}, BuildProps{}, KernelProps{})

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

// Modules implementing defaultable can refer to bob_defaults via the
// `defaults` or `flag_defaults` property
type defaultable interface {
	defaults() []string

	// get properties for which defaults can be applied
	defaultableProperties() []interface{}
}

// Defaults use other defaults, so are themselves `defaultable`
var _ defaultable = (*ModuleDefaults)(nil)

// Defaults have build properties
var _ moduleWithBuildProps = (*ModuleDefaults)(nil)

// Defaults have host and target variants
var _ targetSpecificLibrary = (*ModuleDefaults)(nil)

// Defaults support conditional properties via "features"
var _ Featurable = (*ModuleDefaults)(nil)

// Defaults contain path fragments which need to be prefixes
var _ pathProcessor = (*ModuleDefaults)(nil)

// Defaults support {{match_srcs}} on some properties
var _ matchSourceInterface = (*ModuleDefaults)(nil)

// Defaults have properties that require escaping
var _ propertyEscapeInterface = (*ModuleDefaults)(nil)

var (
	// Map of defaults for each module.
	//
	// This duplicates the information available from Blueprint for
	// each module, but allows us to access the information without
	// having the blueprint.Module available.
	//
	// Populated by DefaultDepsStage1Mutator.
	// Used in DefaultDepsStage2Mutator.
	defaultsMap     = map[string][]string{}
	defaultsMapLock sync.RWMutex
)

// Locally store defaults in defaultsMap
func DefaultDepsStage1Mutator(ctx blueprint.BottomUpMutatorContext) {

	if d, ok := ctx.Module().(*ModuleDefaults); ok {
		srcs := d.getLegacySourceProperties()

		// forbid the use of `srcs` and `exclude_srcs` in `bob_defaults` altogether
		if len(srcs.Srcs) > 0 || len(srcs.Exclude_srcs) > 0 {
			backend.Get().GetLogger().Warn(warnings.DefaultSrcsWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	if l, ok := ctx.Module().(defaultable); ok {
		defaultsMapLock.Lock()
		defer defaultsMapLock.Unlock()

		defaultsMap[ctx.ModuleName()] = l.defaults()
	}

	if gsc, ok := getGenerateCommon(ctx.Module()); ok {
		if len(gsc.Properties.Flag_defaults) > 0 {
			tgt := gsc.Properties.Target
			if !(tgt == toolchain.TgtTypeHost || tgt == toolchain.TgtTypeTarget) {
				utils.Die("Module %s uses flag_defaults '%v' but has invalid target type '%s'",
					ctx.ModuleName(), gsc.Properties.Flag_defaults, tgt)
			}
		}
	}
}

// Take a single defaults module, and recursively expand it to list
// all the hierarchical defaults it depends on (not including itself).
// It's important that the ordering is maintained.
//
//	      a
//	    /   \
//	   b     c
//	 /  \   /  \
//	d    e f    g
//
// ==> d e b f g c
//
// This function is recursive. To prevent getting into an infinite
// loop on encountering a cycle, we pass a list of already visited
// modules in.
func expandDefault(d string, visited []string) []string {
	var defaults []string
	if len(defaultsMap[d]) > 0 {
		for _, def := range defaultsMap[d] {
			if utils.Find(visited, def) >= 0 {
				utils.Die("Defaults module %s depends upon itself", def)
			}
			defaults = append(defaults, expandDefault(def, append(visited, def))...)
			defaults = append(defaults, def)
		}
	}
	return defaults
}

// Adds dependency links for defaults to all modules (but not defaults
// modules). Rather than creating a dependency hierarchy, flatten the
// hierarchy for each module. This allows us to remove duplication of
// defaults modules, while respecting ordering of defaults specified
// on each module, and between hierarchies. Without flattening the
// hierarchy we would need more control over the module visitation
// order in WalkDeps.
func DefaultDepsStage2Mutator(ctx blueprint.BottomUpMutatorContext) {

	_, isDefaults := ctx.Module().(*ModuleDefaults)
	if isDefaults {
		return
	}

	if _, ok := ctx.Module().(defaultable); ok {

		// Get a flattened list of the default hierarchy
		flattenedDefaults := expandDefault(ctx.ModuleName(), []string{})

		var defaults []string

		// Remove duplicates. Defaults that are later in the list
		// override those earlier in the list, so keep the last
		// occurrence of each default.
		for i, el := range flattenedDefaults {
			if utils.Find(flattenedDefaults[i+1:], el) == -1 {
				defaults = append(defaults, el)
			}
		}

		ctx.AddDependency(ctx.Module(), tag.DefaultTag, defaults...)
	}
}
