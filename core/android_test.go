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
)

func checkSplit(t *testing.T, path, expectedBase, expectedRel string) {
	base, rel := splitAndroidPath(path)

	assert.Equal(t, base, expectedBase)
	assert.Equal(t, rel, expectedRel)
}

func Test_splitAndroidPath(t *testing.T) {
	checkSplit(t, "vendor/lib/", "vendor/lib", "")
	checkSplit(t, "vendor/bin/binsubdir", "vendor/bin", "binsubdir")
	checkSplit(t, "vendor/bin", "vendor/bin", "")
	checkSplit(t, "data/nativetest/mytests", "data", "nativetest/mytests")
	checkSplit(t, "$(TARGET_OUT_DATA)/nativetest", "$(TARGET_OUT_DATA)", "nativetest")
	checkSplit(t, "$(TARGET_OUT_VENDOR)/lib", "$(TARGET_OUT_VENDOR)/lib", "")
	checkSplit(t, "$(TARGET_OUT_EXECUTABLES)", "$(TARGET_OUT_EXECUTABLES)", "")
	checkSplit(t, "$(TARGET_OUT_SHARED_LIBRARIES)/libdir", "$(TARGET_OUT_SHARED_LIBRARIES)", "libdir")
}
