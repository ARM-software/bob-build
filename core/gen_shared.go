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

	"github.com/ARM-software/bob-build/abstr"
)

type generateSharedLibrary struct {
	generateLibrary
}

//// Support GenerateLibraryInterface

func (m *generateSharedLibrary) libExtension() string {
	return ".so"
}

//// Support PhonyInterface, DependentInterface

// List of everything generated by this target
func (m *generateSharedLibrary) outputs(g generatorBackend) []string {
	return []string{getLibraryGeneratedPath(m, g)}
}

//// Support Installable

func (m *generateSharedLibrary) filesToInstall(ctx abstr.ModuleContext, g generatorBackend) []string {
	return []string{getLibraryGeneratedPath(m, g)}
}

//// Support blueprint.Module

func (m *generateSharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		inouts := inouts(m, ctx, g)
		g.genSharedActions(m, ctx, inouts)
	}
}

//// Factory functions

func genSharedLibFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &generateSharedLibrary{}
	module.generateCommon.Properties.Features.Init(config.getAvailableFeatures(), GenerateProps{},
		GenerateLibraryProps{})
	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.generateCommon.Properties,
		&module.Properties,
	}
}
