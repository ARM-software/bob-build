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
	"github.com/ARM-software/bob-build/core/file"
	"github.com/google/blueprint"
)

type ModuleStaticLibrary struct {
	ModuleLibrary
}

func (m *ModuleStaticLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).staticActions(m, ctx)
	}
}

func (m *ModuleStaticLibrary) outputFileName() string {
	return m.outputName() + ".a"
}

func (m ModuleStaticLibrary) GetProperties() interface{} {
	return m.ModuleLibrary.Properties
}

func (m *ModuleStaticLibrary) OutFiles() (srcs file.Paths) {
	fp := file.NewPath(m.outputFileName(), string(m.getTarget()), file.TypeArchive) // TODO: refactor outputs() to use file.Paths
	srcs = srcs.AppendIfUnique(fp)
	return
}

func (m *ModuleStaticLibrary) OutFileTargets() []string {
	return []string{}
}

func staticLibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleStaticLibrary{}
	return module.LibraryFactory(config, module)
}
