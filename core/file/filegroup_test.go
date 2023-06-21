/*
 * Copyright 2018-2020, 2023 Arm Limited.
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

package file

import (
	"testing"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/stretchr/testify/assert"
)

func TestLinuxGroups(t *testing.T) {

	backend.Setup(config.GetEnvironmentVariables(),
		config.CreateMockConfig(map[string]interface{}{
			"builder_ninja": true,
		}),
		nil, // logger is nil here, not used in these tests
	)

	paths := Paths{
		NewPath("file0", "0", TypeUnset),
		NewPath("file1", "1", TypeGenerated),
		NewPath("file2", "2", TypeTool),
		NewPath("file3", "3", TypeBinary),
		NewPath("file4", "4", TypeExecutable),
		NewPath("file5", "5", TypeImplicit),
		NewPath("file.c", "6", TypeUnset),
		NewPath("file.cpp", "7", TypeUnset),
		NewPath("file.s", "8", TypeUnset),
		NewPath("file.h", "9", TypeUnset),
		NewPath("file.a", "10", TypeUnset),
		NewPath("file.so", "11", TypeUnset),
		NewPath("file.c", "12", TypeCompilable),
	}

	t.Run("ToStringSliceBuildPaths", func(t *testing.T) {
		expected := []string{
			"${SrcDir}/file0",
			"${BuildDir}/gen/1/file1",
			"${SrcDir}/file2",
			"${BuildDir}/3/executable/file3",
			"${BuildDir}/4/executable/file4",
			"${SrcDir}/file5",
			"${SrcDir}/file.c",
			"${SrcDir}/file.cpp",
			"${SrcDir}/file.s",
			"${SrcDir}/file.h",
			"${BuildDir}/10/static/file.a",
			"${BuildDir}/11/shared/file.so",
			"${SrcDir}/file.c",
		}

		assert.Equal(t, expected, paths.ToStringSlice(
			func(p Path) string {
				return p.BuildPath()
			}))
	})

	t.Run("ToStringSliceBuildPathsFiltered", func(t *testing.T) {
		expected := []string{
			"${BuildDir}/10/static/file.a",
		}

		assert.Equal(t, expected, paths.ToStringSliceIf(
			func(p Path) bool {
				return p.IsType(TypeArchive)
			},
			func(p Path) string {
				return p.BuildPath()
			}))
	})

}
