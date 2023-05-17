/*
 * Copyright 2018-2023 Arm Limited.
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
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"

	"github.com/google/blueprint"
)

type ModuleFilegroup struct {
	module.ModuleBase
	Properties struct {
		SourceProps
		Features
	}
}

// All interfaces supported by filegroup
type filegroupInterface interface {
	pathProcessor
	FileResolver
	FileProvider
}

var _ filegroupInterface = (*ModuleFilegroup)(nil) // impl check

func (m *ModuleFilegroup) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.ResolveFiles(ctx, g)
}

func (m *ModuleFilegroup) OutFiles(g generatorBackend) file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleFilegroup) OutFileTargets() []string {
	return m.Properties.GetTargets()
}

func (m *ModuleFilegroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getBackend(ctx).filegroupActions(m, ctx)
}

func (m *ModuleFilegroup) shortName() string {
	return m.Name()
}

func (m *ModuleFilegroup) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.SourceProps.processPaths(ctx, g)
}

func (m *ModuleFilegroup) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.SourceProps,
	}
}

func (m *ModuleFilegroup) Features() *Features {
	return &m.Properties.Features
}

func (m ModuleFilegroup) GetProperties() interface{} {
	return m.Properties
}

func filegroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleFilegroup{}
	module.Properties.Features.Init(&config.Properties, SourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
