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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ARM-software/bob-build/internal/utils"
)

func checkSplit(t *testing.T, path, expectedBase, expectedRel string) {
	base, rel := splitAndroidPath(path)

	assert.Equal(t, base, expectedBase)
	assert.Equal(t, rel, expectedRel)
}

func Test_splitAndroidPath(t *testing.T) {
	checkSplit(t, "bin/binsubdir", "bin", "binsubdir")
	checkSplit(t, "lib/egl", "lib", "egl")
	checkSplit(t, "etc/firmware", "etc/firmware", "")
	checkSplit(t, "firmware/subdir", "firmware", "subdir")
	checkSplit(t, "etc/subdir", "etc", "subdir")
	checkSplit(t, "tests/subdir", "tests", "subdir")
	checkSplit(t, "data/nativetest/mytests", "data/nativetest", "mytests")
	checkSplit(t, "$(TARGET_OUT_DATA)/nativetest", "$(TARGET_OUT_DATA)/nativetest", "")
	checkSplit(t, "$(TARGET_OUT_DATA_NATIVE_TEST)/mytests", "$(TARGET_OUT_DATA_NATIVE_TEST)", "mytests")
	checkSplit(t, "$(TARGET_OUT_VENDOR)/lib", "$(TARGET_OUT_VENDOR)/lib", "")
	checkSplit(t, "$(TARGET_OUT_EXECUTABLES)", "$(TARGET_OUT_EXECUTABLES)", "")
	checkSplit(t, "$(TARGET_OUT_SHARED_LIBRARIES)/libdir", "$(TARGET_OUT_SHARED_LIBRARIES)", "libdir")
	checkSplit(t, "unknown/path", "unknown/path", "")
}

// Ensure that every translatable Android.mk variable and its translation have
// a corresponding entry in androidInstallLocationSplits.
func Test_androidMkTranslations(t *testing.T) {
	checkHasSplit := func(path string) {
		_, ok := findAndroidInstallLocationSplit(utils.SplitPath(path))
		assert.True(t, ok, "Could not find split for '"+path+"'")
	}

	for mkVar, soongPath := range androidMkInstallLocationTranslations {
		mkVar = "$(" + mkVar + ")"
		checkHasSplit(mkVar)
		if soongPath != "" {
			// TARGET_OUT maps to an empty soong path "", which
			// can't have a matching split entry.
			checkHasSplit(soongPath)
		}
		assert.Equal(t, soongPath, expandAndroidMkInstallVars(mkVar))
	}
}
