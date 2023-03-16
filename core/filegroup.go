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
	"github.com/google/blueprint"
)

type filegroup struct {
	moduleBase
	Properties struct {
		SourceProps
		Features
	}
}

// All interfaces supported by filegroup
type filegroupInterface interface {
	pathProcessor
	FileResolver
	SourceFileProvider
}

var _ filegroupInterface = (*filegroup)(nil) // impl check

func (m *filegroup) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.ResolveFiles(ctx, g)
}

func (m *filegroup) OutSrcs() FilePaths {
	return m.Properties.GetDirectSrcs()
}

func (m *filegroup) OutSrcTargets() []string {
	return m.Properties.GetSrcTargets()
}

func (m *filegroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getBackend(ctx).filegroupActions(m, ctx)
}

func (m *filegroup) shortName() string {
	return m.Name()
}

func (m *filegroup) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.SourceProps.processPaths(ctx, g)
}

func (m *filegroup) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.SourceProps,
	}
}

func (m *filegroup) Features() *Features {
	return &m.Properties.Features
}

func (m filegroup) GetProperties() interface{} {
	return m.Properties
}

func filegroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &filegroup{}
	module.Properties.Features.Init(&config.Properties, SourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
