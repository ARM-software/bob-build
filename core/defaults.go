/*
 * Copyright 2018-2021 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"fmt"
	"sync"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/utils"
)

type defaults struct {
	moduleBase

	Properties struct {
		Features
		Build
		KernelProps
		// The list of default properties that should prepended to all configuration
		Defaults []string
	}
}

func (m *defaults) supportedVariants() []tgtType {
	return []tgtType{tgtTypeHost, tgtTypeTarget}
}

func (m *defaults) disable() {
	panic("disable() called on Default")
}

func (m *defaults) setVariant(variant tgtType) {
	m.Properties.TargetType = variant
}

func (m *defaults) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *defaults) defaults() []string {
	return m.Properties.Defaults
}

func (m *defaults) build() *Build {
	return &m.Properties.Build
}

func (m *defaults) defaultableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.KernelProps,
	}
}

func (m *defaults) featurableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.KernelProps,
	}
}

func (m *defaults) targetableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.KernelProps,
	}
}

func (m *defaults) features() *Features {
	return &m.Properties.Features
}

func (m *defaults) getTarget() tgtType {
	return m.Properties.TargetType
}

func (m *defaults) getTargetSpecific(variant tgtType) *TargetSpecific {
	return m.Properties.getTargetSpecific(variant)
}

func (m *defaults) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.Build.processPaths(ctx, g)
	m.Properties.KernelProps.processPaths(ctx)
}

func (m *defaults) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

func (m *defaults) getEscapeProperties() []*[]string {
	return []*[]string{
		&m.Properties.Asflags,
		&m.Properties.Cflags,
		&m.Properties.Conlyflags,
		&m.Properties.Cxxflags,
		&m.Properties.Ldflags}
}

func (m *defaults) getSourceProperties() *SourceProps {
	return &m.Properties.SourceProps
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `BuildProps` this is:
//  - Ldflags
//  - Cflags
//  - Conlyflags
//  - Cxxflags
func (m *defaults) getMatchSourcePropNames() []string {
	return []string{"Ldflags", "Cflags", "Conlyflags", "Cxxflags"}
}

func defaultsFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &defaults{}

	module.Properties.Features.Init(&config.Properties, CommonProps{}, BuildProps{}, KernelProps{}, SplittableProps{})
	module.Properties.Host.init(&config.Properties, CommonProps{}, BuildProps{}, KernelProps{})
	module.Properties.Target.init(&config.Properties, CommonProps{}, BuildProps{}, KernelProps{})

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

var defaultDepTag = dependencyTag{name: "default"}

// Modules implementing defaultable can refer to bob_defaults via the
// `defaults` or `flag_defaults` property
type defaultable interface {
	defaults() []string

	// get properties for which defaults can be applied
	defaultableProperties() []interface{}
}

// Defaults use other defaults, so are themselves `defaultable`
var _ defaultable = (*defaults)(nil)

// Defaults have build properties
var _ moduleWithBuildProps = (*defaults)(nil)

// Defaults have host and target variants
var _ targetSpecificLibrary = (*defaults)(nil)

// Defaults support conditional properties via "features"
var _ featurable = (*defaults)(nil)

// Defaults contain path fragments which need to be prefixes
var _ pathProcessor = (*defaults)(nil)

// Defaults support {{match_srcs}} on some properties
var _ matchSourceInterface = (*defaults)(nil)

// Defaults have properties that require escaping
var _ propertyEscapeInterface = (*defaults)(nil)

var (
	// Map of defaults for each module.
	//
	// This duplicates the information available from Blueprint for
	// each module, but allows us to access the information without
	// having the blueprint.Module available.
	//
	// Populated by defaultDepsStage1Mutator.
	// Used in defaultDepsStage2Mutator.
	defaultsMap     = map[string][]string{}
	defaultsMapLock sync.RWMutex
)

// Locally store defaults in defaultsMap
func defaultDepsStage1Mutator(mctx blueprint.BottomUpMutatorContext) {

	if l, ok := mctx.Module().(defaultable); ok {
		defaultsMapLock.Lock()
		defer defaultsMapLock.Unlock()

		defaultsMap[mctx.ModuleName()] = l.defaults()
	}

	if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		if len(gsc.Properties.Flag_defaults) > 0 {
			tgt := gsc.Properties.Target
			if !(tgt == tgtTypeHost || tgt == tgtTypeTarget) {
				panic(fmt.Errorf("Module %s uses flag_defaults '%v' but has invalid target type '%s'",
					mctx.ModuleName(), gsc.Properties.Flag_defaults, tgt))
			}
		}
	}
}

// Take a single defaults module, and recursively expand it to list
// all the hierarchical defaults it depends on (not including itself).
// It's important that the ordering is maintained.
//
//        a
//      /   \
//    b       c
//  /  \     /  \
// d    e   f    g
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
				panic(fmt.Errorf("Defaults module %s depends upon itself", def))
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
func defaultDepsStage2Mutator(mctx blueprint.BottomUpMutatorContext) {

	_, isDefaults := mctx.Module().(*defaults)
	if isDefaults {
		return
	}

	if _, ok := mctx.Module().(defaultable); ok {

		// Get a flattened list of the default hierarchy
		flattenedDefaults := expandDefault(mctx.ModuleName(), []string{})

		var defaults []string

		// Remove duplicates. Defaults that are later in the list
		// override those earlier in the list, so keep the last
		// occurrence of each default.
		for i, el := range flattenedDefaults {
			if utils.Find(flattenedDefaults[i+1:], el) == -1 {
				defaults = append(defaults, el)
			}
		}

		mctx.AddDependency(mctx.Module(), defaultDepTag, defaults...)
	}
}
