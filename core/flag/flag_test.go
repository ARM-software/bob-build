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

func TestLinux(t *testing.T) {

	backend.Setup(config.GetEnvironmentVariables(),
		config.CreateMockConfig(map[string]interface{}{
			"builder_ninja": true,
		}),
		nil, // logger is nil here, not used in these tests
	)

	raw_local_path := "local/foo"
	raw_global_path := "/global/foo"

	t.Run("SimpleFlag", func(t *testing.T) {
		f := FromString("-Wall", TypeTransitive)
		assert.Equal(t, "-Wall", f.ToString())

		assert.True(t, f.IsType(TypeTransitive)) // matches exactly

		assert.True(t, f.MatchesType(TypeTransitive|TypeExported)) //loosely matches

		assert.True(t, f.IsNotType(TypeExported))
		assert.True(t, f.IsNotType(TypeExported|TypeTransitive))

		assert.Equal(t, f.Raw(), "-Wall")
		assert.Equal(t, f.Raw(), f.ToString()) //Simple case raw == string
	})

	t.Run("FromIncludePath", func(t *testing.T) {
		tag := TypeIncludeLocal
		f := FromIncludePath(raw_local_path, tag)
		assert.Equal(t, "-I${SrcDir}/local/foo", f.ToString())
		assert.Equal(t, f.Type(), tag|TypeInclude)
		assert.Equal(t, f.Raw(), raw_local_path)

		tag |= TypeIncludeSystem
		f = FromIncludePath(raw_local_path, tag)
		assert.Equal(t, "-isystem ${SrcDir}/local/foo", f.ToString())

		tag ^= TypeIncludeLocal
		f = FromIncludePath(raw_global_path, tag)
		assert.Equal(t, "-isystem /global/foo", f.ToString())

		tag ^= TypeIncludeSystem
		f = FromIncludePath(raw_global_path, tag)
		assert.Equal(t, "-I/global/foo", f.ToString())
	})
}
