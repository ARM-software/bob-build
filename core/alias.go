/*
 * Copyright 2018-2019 Arm Limited.
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
	"github.com/google/blueprint"
)

var aliasTag = dependencyTag{name: "alias"}

// Modules implementing the aliasable interface can be referenced by a
// bob_alias module
type aliasable interface {
	getAliasList() []string
}

// AliasableProps are embedded in modules which can be aliased
type AliasableProps struct {
	// Adds this module to an alias
	Add_to_alias []string
}

func (p *AliasableProps) getAliasList() []string {
	return p.Add_to_alias
}

// AliasProps describes the properties of the bob_alias module
type AliasProps struct {
	// Modules that this alias will cause to build
	Srcs []string
	AliasableProps
}

// Type representing each bob_alias module
type alias struct {
	moduleBase
	Properties struct {
		AliasProps
		Features
	}
}

func (m *alias) features() *Features {
	return &m.Properties.Features
}

func (m *alias) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.AliasProps}
}

func (m *alias) getAliasList() []string {
	return m.Properties.getAliasList()
}

// Called by Blueprint to generate the rules associated with the alias.
// This is forwarded to the backend to handle.
func (m *alias) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getBackend(ctx).aliasActions(m, ctx)
}

// Create the structure representing the bob_alias
func aliasFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &alias{}
	module.Properties.Features.Init(config.getAvailableFeatures(), AliasProps{})
	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

// Setup dependencies between aliases and their targets
func aliasMutator(mctx blueprint.BottomUpMutatorContext) {
	if a, ok := mctx.Module().(*alias); ok {
		parseAndAddVariationDeps(mctx, aliasTag, a.Properties.Srcs...)
	}
	if a, ok := mctx.Module().(aliasable); ok {
		for _, s := range a.getAliasList() {
			mctx.AddReverseDependency(mctx.Module(), aliasTag, s)
		}
	}
}
