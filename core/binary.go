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
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/google/blueprint"
)

type ModuleBinary struct {
	ModuleLibrary
}

// binary supports:
type binaryInterface interface {
	stripable
	linkableModule
	FileProvider // A binary can provide itself as a source
}

var _ binaryInterface = (*ModuleBinary)(nil) // impl check

func (m *ModuleBinary) outputs() []string {
	return m.outs
}

func (m *ModuleBinary) OutFiles() (srcs file.Paths) {
	for _, out := range m.outputs() {
		// TODO: Remove the need to know the buildDir here, this prevents the removal of generatorBackend
		fp := file.NewPath(strings.TrimPrefix(out, backend.Get().BuildDir()), "", file.TypeBinary|file.TypeExecutable) // TODO: refactor outputs() to use file.Paths
		srcs = srcs.AppendIfUnique(fp)
	}
	return
}

func (m *ModuleBinary) OutFileTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleBinary) strip() bool {
	return m.Properties.Strip != nil && *m.Properties.Strip
}

func (m *ModuleBinary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).binaryActions(m, ctx)
	}
}

func (m *ModuleBinary) outputFileName() string {
	return m.outputName()
}

func (m ModuleBinary) GetProperties() interface{} {
	return m.ModuleLibrary.Properties
}

func binaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleBinary{}
	return module.LibraryFactory(config, module)
}
