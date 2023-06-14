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

package flag

import (
	"github.com/google/blueprint"
)

type Consumer interface {
	// Returns flags consumed by this module, listed by itself
	FlagsIn() Flags

	// Returns flags consumed by this module, including any flags provided by its upstream providers.
	FlagsInTransitive(blueprint.BaseModuleContext) Flags
}

// Basic common implementation, certain targets may wish to customize this.
func ReferenceFlagsInTransitive(ctx blueprint.BaseModuleContext) (ret Flags) {

	//There are a few possible ways for flags to be propagated. In legacy targets there are:
	// * Direct dependancies
	// * Indirect dependancies via reexport_libs
	// In new targets there are also:
	// * Transitive exports. These make `reexport_libs` obsolete.

	visited := map[string]bool{}

	ctx.VisitDirectDeps(func(child blueprint.Module) {
		if visited[child.Name()] {
			return
		}
		visited[child.Name()] = true

		if provider, ok := child.(Provider); ok {
			flags := provider.FlagsOut().Filtered(
				func(f Flag) bool {
					return f.MatchesType(TypeExported | TypeTransitive)
				},
			)

			flags.ForEach(func(f Flag) bool {
				ret = ret.AppendIfUnique(f)
				return true
			})
		}
	})

	// Handle transitive flags. Any previously visited module should be skipped as Transitive flags are normally exported as well.
	ctx.WalkDeps(func(parent, child blueprint.Module) bool {
		if visited[child.Name()] {
			return true
		}
		visited[child.Name()] = true
		if provider, ok := child.(Provider); ok {
			flags := provider.FlagsOut().Filtered(
				func(f Flag) bool {
					return f.IsType(TypeTransitive)
				},
			)

			flags.ForEach(func(f Flag) bool {
				ret = ret.AppendIfUnique(f)
				return true
			})
		}

		return true
	})

	return
}
