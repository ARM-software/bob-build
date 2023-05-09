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

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_basic_filepaths_tests(t *testing.T) {
	fp1 := sourceFilePath{"somemod/foo.c", "somemod", "${SrcDir}"}
	fp2 := generatedFilePath{"${BuildDir}/gen/somemod/file.ext", "somemod/file.ext", "${BuildDir}/gen/somemod"}
	fp3 := generatedFilePath{"${BuildDir}/gen/othermod/file.ext", "othermod/file.ext", "${BuildDir}/gen/othermod"}

	fps := FilePaths{
		fp1, fp2,
	}

	other := FilePaths{fp3}

	assert.True(t, fps.Contains(fp1), "Contains returns true for exisiting path")
	assert.False(t, fps.Contains(fp3), "Contains returns false for missing path")

	// Paths
	assert.Equal(t, "${BuildDir}/gen/somemod/file.ext", fp2.buildPath(), "Correct build path returned for generated file.")
	assert.Equal(t, "somemod/file.ext", fp2.localPath(), "Correct local path returned for generated file.")
	assert.Equal(t, "${BuildDir}/gen/somemod", fp2.moduleDir(), "Correct module dir path returned for generated file.")

	assert.Equal(t, "${SrcDir}/somemod/foo.c", fp1.buildPath(), "Correct build path returned for static file.")
	assert.Equal(t, "somemod/foo.c", fp1.localPath(), "Correct local path returned for static file.")
	assert.Equal(t, "${SrcDir}/somemod", fp1.moduleDir(), "Correct module dir path returned for static file.")

	merged := fps.Merge(other)
	assert.True(t, merged.Contains(fp1), "Check merge operation")
	assert.True(t, merged.Contains(fp2), "Check merge operation")
	assert.True(t, merged.Contains(fp3), "Check merge operation")

	srcs_count := 0
	gen_count := 0
	all_files := 0

	for s := range merged.IteratePredicate(func(fp filePath) bool {
		_, isSource := fp.(sourceFilePath)
		return isSource
	}) {
		_, isSource := s.(sourceFilePath)
		assert.True(t, isSource, "Source predicate expected")
		srcs_count += 1
	}

	for s := range merged.IteratePredicate(func(fp filePath) bool {
		_, isGen := fp.(generatedFilePath)
		return isGen
	}) {
		_, isGen := s.(generatedFilePath)
		assert.True(t, isGen, "Generated predicate expected")
		gen_count += 1
	}

	merged.ForEach(func(fp filePath) bool {
		all_files += 1
		return true
	})

	assert.Equal(t, 2, gen_count, "Correct number of generated files counted in the merged list.")
	assert.Equal(t, 1, srcs_count, "Correct number of sources counted in the merged list.")
	assert.Equal(t, 3, all_files, "Correct number of all files in the merged list.")
}