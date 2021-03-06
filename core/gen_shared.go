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
	"github.com/google/blueprint"
)

type generateSharedLibrary struct {
	generateLibrary
	fileNameExtension string
}

// Verify that the following interfaces are implemented
var _ generateLibraryInterface = (*generateSharedLibrary)(nil)
var _ singleOutputModule = (*generateSharedLibrary)(nil)
var _ sharedLibProducer = (*generateSharedLibrary)(nil)
var _ splittable = (*generateSharedLibrary)(nil)
var _ blueprint.Module = (*generateSharedLibrary)(nil)

func (m *generateSharedLibrary) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	return generateLibraryInouts(m, ctx, g, m.Properties.Headers)
}

//// Support generateLibraryInterface

func (m *generateSharedLibrary) libExtension() string {
	return m.fileNameExtension
}

//// Support blueprint.Module

func (m *generateSharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.genSharedActions(m, ctx)
	}
}

//// Support singleOutputModule

func (m *generateSharedLibrary) outputFileName() string {
	return m.altName() + m.libExtension()
}

//// Support sharedLibProducer

func (m *generateSharedLibrary) getTocName() string {
	return m.outputFileName() + tocExt
}

//// Factory functions

func genSharedLibFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &generateSharedLibrary{}
	module.generateCommon.init(&config.Properties, GenerateProps{},
		GenerateLibraryProps{})

	if config.Properties.GetBool("osx") {
		module.fileNameExtension = ".dylib"
	} else {
		module.fileNameExtension = ".so"
	}
	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.generateCommon.Properties,
		&module.Properties,
	}
}
