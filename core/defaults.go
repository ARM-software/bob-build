/*
 * Copyright 2018-2020 Arm Limited.
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

	"github.com/google/blueprint"
)

type defaults struct {
	moduleBase

	Properties struct {
		Features
		Build
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
	return &SplittableProps{}
}

func (m *defaults) defaults() []string {
	return m.Properties.Defaults
}

func (m *defaults) build() *Build {
	return &m.Properties.Build
}

func (m *defaults) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.Build.BuildProps, &m.Properties.Build.SplittableProps}
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
func (m *defaults) getMatchSourcePropNames() []string {
	return []string{"Ldflags"}
}

func defaultsFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &defaults{}

	module.Properties.Build.init(&config.Properties)
	module.Properties.Features.Init(&config.Properties, BuildProps{}, SplittableProps{})

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

var defaultDepTag = dependencyTag{name: "default"}

// Modules implementing defaultable can refer to bob_defaults via the
// `defaults` property
type defaultable interface {
	build() *Build
	features() *Features
	defaults() []string
}

// Defaults use other defaults, so are themselves `defaultable`
var _ defaultable = (*defaults)(nil)

// Defaults have build properties
var _ moduleWithBuildProps = (*defaults)(nil)

// Defaults have host and target variants
var _ targetable = (*defaults)(nil)

// Defaults support conditional properties via "features"
var _ featurable = (*defaults)(nil)

// Defaults contain path fragments which need to be prefixes
var _ pathProcessor = (*defaults)(nil)

// Defaults support {{match_srcs}} on some properties
var _ matchSourceInterface = (*defaults)(nil)

// Defaults have properties that require escaping
var _ propertyEscapeInterface = (*defaults)(nil)

func defaultDepsMutator(mctx blueprint.BottomUpMutatorContext) {
	if l, ok := mctx.Module().(defaultable); ok {
		mctx.AddDependency(mctx.Module(), defaultDepTag, l.defaults()...)
	}
	if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		if len(gsc.Properties.Flag_defaults) > 0 {
			tgt := gsc.Properties.Target
			if !(tgt == tgtTypeHost || tgt == tgtTypeTarget) {
				panic(fmt.Errorf("Module %s uses flag_defaults '%v' but has invalid target type '%s'",
					mctx.ModuleName(), gsc.Properties.Flag_defaults, tgt))
			}
			mctx.AddDependency(mctx.Module(), defaultDepTag, gsc.Properties.Flag_defaults...)
		}
	}
}
