/*
 * Copyright 2023 Arm Limited.
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
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type ModuleStrictGenerateCommon struct {
	module.ModuleBase
	Properties struct {
		EnableableProps
		Features
		StrictGenerateProps
	}
	deps []string
}

var _ FileConsumer = (*ModuleStrictGenerateCommon)(nil)

func (m *ModuleStrictGenerateCommon) init(properties *config.Properties, list ...interface{}) {
	m.Properties.Features.Init(properties, list...)
}

func (m *ModuleStrictGenerateCommon) processPaths(ctx blueprint.BaseModuleContext) {
	m.deps = utils.MixedListToBobTargets(m.Properties.StrictGenerateProps.Tool_files)
	m.Properties.StrictGenerateProps.processPaths(ctx)
}

func (m *ModuleStrictGenerateCommon) GetTargets() []string {
	return m.Properties.GetTargets()
}

func (m *ModuleStrictGenerateCommon) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.GetFiles(ctx)
}

func (m *ModuleStrictGenerateCommon) GetDirectFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleStrictGenerateCommon) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleStrictGenerateCommon) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleStrictGenerateCommon) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.EnableableProps, &m.Properties.StrictGenerateProps}
}

func (m *ModuleStrictGenerateCommon) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

// Module implementing `StrictGenerator`
// are able to generate output files
type StrictGenerator interface {
	getStrictGenerateCommon() *ModuleStrictGenerateCommon
}

func getStrictGenerateCommon(i interface{}) (*ModuleStrictGenerateCommon, bool) {
	var m *ModuleStrictGenerateCommon
	sg, ok := i.(StrictGenerator)
	if ok {
		m = sg.getStrictGenerateCommon()
	}
	return m, ok
}
