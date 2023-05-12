/*
 * Copyright 2019-2020, 2023 Arm Limited.
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
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

type StripProps struct {
	// When set, strip symbols and debug information from libraries
	// and binaries. This is a separate stage that occurs after
	// linking and before post install.
	//
	// On Android, its infrastructure is used to do the stripping. If
	// not enabled, follow Android's default behaviour.
	Strip *bool

	// Module specifying a directory for debug information
	Debug_info *string

	// The path retrieved from debug install group so we don't need to
	// walk dependencies to get it
	Debug_path *string `blueprint:"mutated"`
}

func (props *StripProps) getDebugInfo() *string {
	return props.Debug_info
}

func (props *StripProps) getDebugPath() *string {
	return props.Debug_path
}

func (props *StripProps) setDebugPath(path *string) {
	props.Debug_path = path
}

type stripable interface {
	strip() bool
	getTarget() toolchain.TgtType
	stripOutputDir(g generatorBackend) string

	getDebugInfo() *string
	getDebugPath() *string
	setDebugPath(*string)
}

func debugInfoMutator(ctx blueprint.TopDownMutatorContext) {
	if m, ok := ctx.Module().(stripable); ok {
		path := getInstallGroupPathFromTag(ctx, DebugInfoTag)
		m.setDebugPath(path)
	}
}
