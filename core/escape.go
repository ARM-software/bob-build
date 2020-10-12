/*
 * Copyright 2020 Arm Limited.
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

	"github.com/ARM-software/bob-build/internal/escape"
)

type propertyEscapeInterface interface {
	getEscapeProperties() []*[]string
}

func escapeMutator(mctx blueprint.TopDownMutatorContext) {
	// This mutator is not registered on the androidbp backend, as it
	// doesn't need escaping

	g := getBackend(mctx)
	module := mctx.Module()

	if _, ok := module.(*defaults); ok {
		// No need to apply to defaults
		return
	}

	if e, ok := module.(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, skip execution
			return
		}
	}

	// Escape libraries as well as generator modules
	if m, ok := module.(propertyEscapeInterface); ok {
		escapeProps := m.getEscapeProperties()

		for _, prop := range escapeProps {
			// If the flags contain template sequences, we avoid escaping those
			*prop = escape.EscapeTemplatedStringList(*prop, g.escapeFlag)
		}
	}
}
