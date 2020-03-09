/*
 * Copyright 2019-2020 Arm Limited.
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

type ExternalLibProps struct {
	Export_cflags  []string
	Export_ldflags []string
	Ldlibs         []string
}

type externalLib struct {
	moduleBase
	Properties struct {
		ExternalLibProps
		Features
	}
}

func (m *externalLib) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.ExternalLibProps}
}

func (m *externalLib) features() *Features {
	return &m.Properties.Features
}

func (m *externalLib) outputName() string   { return m.Name() }
func (m *externalLib) altName() string      { return m.outputName() }
func (m *externalLib) altShortName() string { return m.altName() }
func (m *externalLib) shortName() string    { return m.Name() }

// External libraries have no outputs - they are already built.
func (m *externalLib) outputs() []string         { return []string{} }
func (m *externalLib) implicitOutputs() []string { return []string{} }

// Implement the splittable interface so "normal" libraries can depend on external ones.
func (m *externalLib) supportedVariants() []tgtType         { return []tgtType{tgtTypeHost, tgtTypeTarget} }
func (m *externalLib) disable()                             {}
func (m *externalLib) setVariant(tgtType)                   {}
func (m *externalLib) getSplittableProps() *SplittableProps { return &SplittableProps{} }

// Implement the propertyExporter interface so that external libraries can pass
// on properties e.g. from pkg-config

func (m *externalLib) exportCflags() []string           { return m.Properties.Export_cflags }
func (m *externalLib) exportIncludeDirs() []string      { return []string{} }
func (m *externalLib) exportLocalIncludeDirs() []string { return []string{} }
func (m *externalLib) exportLdflags() []string          { return m.Properties.Export_ldflags }
func (m *externalLib) exportLdlibs() []string           { return m.Properties.Ldlibs }
func (m *externalLib) exportSharedLibs() []string       { return []string{} }

var _ propertyExporter = (*externalLib)(nil)

// External libraries have no actions - they are already built.
func (m *externalLib) GenerateBuildActions(ctx blueprint.ModuleContext) {}

func externalLibFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &externalLib{}
	module.Properties.Features.Init(&config.Properties, ExternalLibProps{})
	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
