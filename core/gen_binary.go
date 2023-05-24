/*
 * Copyright 2018-2021, 2023 Arm Limited.
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
	"github.com/google/blueprint"
)

type generateBinary struct {
	generateLibrary
}

// Verify that the following interfaces are implemented
var _ FileProvider = (*generateBinary)(nil)
var _ generateLibraryInterface = (*generateBinary)(nil)
var _ singleOutputModule = (*generateBinary)(nil)
var _ splittable = (*generateBinary)(nil)
var _ blueprint.Module = (*generateBinary)(nil)

func (m *generateBinary) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	return generateLibraryInouts(m, ctx, g, m.Properties.Headers)
}

func (m *generateBinary) OutFiles(g generatorBackend) (files file.Paths) {
	// TODO: implement me
	return
}

//// Support generateLibraryInterface

func (m *generateBinary) libExtension() string {
	return ""
}

//// Support singleOutputModule

func (m *generateBinary) outputFileName() string {
	return m.altName() + m.libExtension()
}

//// Support blueprint.Module

func (m *generateBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).genBinaryActions(m, ctx)
	}
}

func (m generateBinary) GetProperties() interface{} {
	return m.generateLibrary.Properties
}

//// Factory functions

func genBinaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateBinary{}
	module.ModuleGenerateCommon.init(&config.Properties, GenerateProps{})

	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.ModuleGenerateCommon.Properties,
		&module.Properties,
	}
}
