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
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

// CommonProps defines a set of properties which are common
// for multiple module types.
type CommonProps struct {
	LegacySourceProps
	IncludeDirsProps
	InstallableProps
	EnableableProps
	AndroidProps
	AliasableProps

	// Flags used for C compilation
	Cflags []string
}

func (c *CommonProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)

	c.LegacySourceProps.processPaths(ctx, g)
	c.InstallableProps.processPaths(ctx, g)
	c.IncludeDirsProps.Local_include_dirs = utils.PrefixDirs(c.IncludeDirsProps.Local_include_dirs, prefix)
}
