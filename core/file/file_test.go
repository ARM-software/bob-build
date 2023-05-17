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

	"github.com/stretchr/testify/assert"
)

func Test_basic_paths_tests(t *testing.T) {
	// TODO: fix collections
	fp1 := Path{"${SrcDir}", "somemod/foo.c", "${SrcDir}/somemod/foo.c", TypeC}
	fp2 := Path{"${BuildDir}/gen/somemod", "file.ext", "${BuildDir}/gen/somemod/file.ext", TypeGenerated}
	fp3 := Path{"${BuildDir}/gen/othermod", "file.ext", "${BuildDir}/gen/othermod/file.ext", TypeGenerated}

	fps := Paths{
		fp1, fp2,
	}

	other := Paths{fp3}

	assert.True(t, fps.Contains(fp1), "Contains returns true for exisiting path")
	assert.False(t, fps.Contains(fp3), "Contains returns false for missing path")

	merged := fps.Merge(other)
	assert.True(t, merged.Contains(fp1), "Check merge operation")
	assert.True(t, merged.Contains(fp2), "Check merge operation")
	assert.True(t, merged.Contains(fp3), "Check merge operation")

	srcs_count := 0
	gen_count := 0
	all_files := 0

	for s := range merged.IteratePredicate(func(fp Path) bool {
		return fp.IsType(TypeCompilable)
	}) {
		isSource := s.IsType(TypeCompilable)
		assert.True(t, isSource, "Source predicate expected")
		srcs_count += 1
	}

	for s := range merged.IteratePredicate(func(fp Path) bool {
		return fp.IsType(TypeGenerated)
	}) {
		isGen := s.IsType(TypeGenerated)
		assert.True(t, isGen, "Generated predicate expected")
		gen_count += 1
	}

	merged.ForEach(func(fp Path) bool {
		all_files += 1
		return true
	})

	assert.Equal(t, 2, gen_count, "Correct number of generated files counted in the merged list.")
	assert.Equal(t, 1, srcs_count, "Correct number of sources counted in the merged list.")
	assert.Equal(t, 3, all_files, "Correct number of all files in the merged list.")

	assert.Equal(t, true, fp1.IsType(TypeC), "Correct set tag for Path.")
	assert.Equal(t, true, fp1.IsType(TypeCompilable), "Correct mask that is a superset of tag for Path.")
	assert.Equal(t, false, fp1.IsType(TypeCpp), "Correctly report non-matching Type for Path.")
}

func Test_source_path_test(t *testing.T) {
	FactorySetup("${BuildDir}", "${SrcDir}")

	fp := NewPath("moduleFoo/srcs/main.c", FileNoNameSpace, TypeUnset)

	assert.Equal(t, "${SrcDir}/moduleFoo/srcs/main.c", fp.BuildPath())
	assert.Equal(t, "moduleFoo/srcs/main.c", fp.RelBuildPath())
	assert.Equal(t, "moduleFoo/srcs/main.c", fp.ScopedPath())
	assert.Equal(t, "moduleFoo/srcs/main.c", fp.UnScopedPath())
	assert.Equal(t, ".c", fp.Ext())
	assert.True(t, fp.IsType(TypeC))
	assert.True(t, fp.IsNotType(TypeAsm|TypeGenerated))
	assert.True(t, fp.IsNotType(TypeGenerated))
}

func Test_generated_simple(t *testing.T) {
	FactorySetup("${BuildDir}", "${SrcDir}")

	fp := NewPath("foo.c", "bar", TypeGenerated)

	assert.Equal(t, "${BuildDir}/gen/bar/foo.c", fp.BuildPath())
	assert.Equal(t, "gen/bar/foo.c", fp.RelBuildPath())
	assert.Equal(t, "bar/foo.c", fp.ScopedPath()) // foo/bar/transfomr.c
	assert.Equal(t, "foo.c", fp.UnScopedPath())
	assert.Equal(t, ".c", fp.Ext())
	assert.True(t, fp.IsType(TypeC|TypeGenerated))
	assert.True(t, fp.IsNotType(TypeAsm))
	assert.False(t, fp.IsNotType(TypeGenerated))
}
