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

type staticLibrary struct {
	library
}

func (l *staticLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(l) {
		getBackend(ctx).staticActions(l, ctx)
	}
}

func (l *staticLibrary) outputFileName() string {
	return l.outputName() + ".a"
}

func staticLibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &staticLibrary{}
	return module.LibraryFactory(config, module)
}
