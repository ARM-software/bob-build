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
	"testing"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/stretchr/testify/assert"
)

func TestCollections(t *testing.T) {

	backend.Setup(config.GetEnvironmentVariables(),
		config.CreateMockConfig(map[string]interface{}{
			"builder_ninja": true,
		}),
		nil, // logger is nil here, not used in these tests
	)

	flags := Flags{
		FromString("Unset", TypeUnset),
		FromString("Include|     |             |        ", TypeInclude),
		FromString("Include|     |             |Exported", TypeInclude|TypeExported),
		FromString("Include|     |IncludeSystem|        ", TypeInclude|TypeIncludeSystem),
		FromString("Include|     |IncludeSystem|Exported", TypeInclude|TypeIncludeSystem|TypeExported),
		FromString("Include|Local|             |        ", TypeInclude|TypeIncludeLocal),
		FromString("Include|Local|             |Exported", TypeInclude|TypeIncludeLocal|TypeExported),
		FromString("Include|Local|IncludeSystem|        ", TypeInclude|TypeIncludeLocal|TypeIncludeSystem),
		FromString("Include|Local|IncludeSystem|Exported", TypeInclude|TypeIncludeLocal|TypeIncludeSystem|TypeExported),
		FromString("Asm", TypeAsm),
		FromString("C", TypeC),
		FromString("Cpp", TypeCpp),
		FromString("CC", TypeCC),
		FromString("CC|Exported", TypeCC|TypeExported),
		FromString("Linker", TypeLinker),
		FromString("Linker|Exported", TypeLinker|TypeExported),
		FromString("Compilable|Exported", TypeCompilable|TypeExported),
		FromString("Compilable", TypeCompilable),
		FromString("Transitive", TypeTransitive),
		FromString("C|Transitive", TypeC|TypeTransitive),
	}

	t.Run("filters", func(t *testing.T) {
		no_exports := flags.Filtered(
			func(f Flag) bool {
				return !f.MatchesType(TypeExported)
			},
		)

		no_exports.ForEach(func(f Flag) bool {
			assert.False(t, f.IsType(TypeExported))
			return true
		})
	})

	t.Run("tag_groups", func(t *testing.T) {
		grouped := flags.GroupByType(TypeTransitive)
		assert.False(t, grouped[0].MatchesType(TypeTransitive))
		assert.True(t, grouped[len(grouped)-1].MatchesType(TypeTransitive))
	})

}
