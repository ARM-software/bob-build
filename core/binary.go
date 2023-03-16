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
	"github.com/google/blueprint"
)

type binary struct {
	library
}

// binary supports:
type binaryInterface interface {
	stripable
	linkableModule
	SourceFileProvider // A binary can provide itself as a source
}

var _ binaryInterface = (*binary)(nil) // impl check

func (m *binary) OutSrcs() (srcs FilePaths) {
	for _, out := range m.outputs() {
		fp := newGeneratedFilePath(out)
		srcs = srcs.AppendIfUnique(fp)
	}
	return
}

func (m *binary) OutSrcTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (b *binary) strip() bool {
	return b.Properties.Strip != nil && *b.Properties.Strip
}

func (b *binary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(b) {
		getBackend(ctx).binaryActions(b, ctx)
	}
}

func (b *binary) outputFileName() string {
	return b.outputName()
}

func (b binary) GetProperties() interface{} {
	return b.library.Properties
}

func binaryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &binary{}
	return module.LibraryFactory(config, module)
}
