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

// Logic common to the Android.mk and Android.bp backends

import (
	"path/filepath"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/utils"
)

var (
	dummyRule = pctx.StaticRule("dummy",
		blueprint.RuleParams{
			// We don't want this rule to do anything, so just echo the target
			Command:     "echo $out",
			Description: "Dummy rule",
		})
)

func enabledAndRequired(m blueprint.Module) bool {
	if e, ok := m.(enableable); ok {
		if !isEnabled(e) || !isRequired(e) {
			return false
		}
	}
	return true
}

// Map of path prefixes and where to split the path into "base" and "rel" sections, roughly
// corresponding to LOCAL_PATH and LOCAL_MODULE_RELATIVE_PATH/relative_install_path.
var androidInstallLocationSplits = map[string]int{
	// Paths in system and vendor have another component, e.g. `bin` or
	// `lib` - after that, it is all relative.
	"system":                2,
	"vendor":                2,
	"$(TARGET_OUT)":         3,
	"$(TARGET_OUT_PRODUCT)": 2,
	"$(TARGET_OUT_SYSTEM)":  2,
	"$(TARGET_OUT_VENDOR)":  2,

	// Android.mk-specific build dir, which a subdirectory per module type.
	"$(TARGET_OUT_GEN)": 2,

	// Filetype-specific Android.mk variables already include the `lib` or `bin` part.
	"$(TARGET_OUT_DATA_EXECUTABLES)":            1,
	"$(TARGET_OUT_DATA_METRIC_TESTS)":           1,
	"$(TARGET_OUT_DATA_NATIVE_TESTS)":           1,
	"$(TARGET_OUT_DATA_SHARED_LIBRARIES)":       1,
	"$(TARGET_OUT_EXECUTABLES)":                 1,
	"$(TARGET_OUT_OEM_EXECUTABLES)":             1,
	"$(TARGET_OUT_OEM_SHARED_LIBRARIES)":        1,
	"$(TARGET_OUT_OPTIONAL_EXECUTABLES)":        1,
	"$(TARGET_OUT_PRODUCT_EXECUTABLES)":         1,
	"$(TARGET_OUT_PRODUCT_SHARED_LIBRARIES)":    1,
	"$(TARGET_OUT_SHARED_LIBRARIES)":            1,
	"$(TARGET_OUT_VENDOR_EXECUTABLES)":          1,
	"$(TARGET_OUT_VENDOR_OPTIONAL_EXECUTABLES)": 1,
	"$(TARGET_OUT_VENDOR_SHARED_LIBRARIES)":     1,

	// /etc contains subdirs like `firmware` which need to be part of the base path
	"vendor/etc":                         3,
	"system/etc":                         3,
	"etc":                                2,
	"$(TARGET_OUT_DATA_ETC)":             2,
	"$(TARGET_OUT_ETC)":                  2,
	"$(TARGET_OUT_OEM_ETC)":              2,
	"$(TARGET_OUT_PRODUCT_ETC)":          2,
	"$(TARGET_OUT_PRODUCT_SERVICES_ETC)": 2,
	"$(TARGET_OUT_VENDOR_ETC)":           2,

	// /data isn't quite so structured, so put most components in the relative_install_path.
	// Note that $(TARGET_OUT_DATA_EXECUTABLES) etc actually maps to /system, so is handled
	// the same as the other filetype-specific stuff - this just catches anything else.
	"data":               1,
	"$(TARGET_OUT_DATA)": 1,

	// /testcases is unstructured
	"testcases":               1,
	"$(TARGET_OUT_TESTCASES)": 1,
}

func splitAndroidPath(path string) (string, string) {
	components := utils.SplitPath(path)

	// If no match, the whole path is the "base" section.
	relStart := len(components)

	// Try longer sections of path first to avoid incorrect matches on common prefixes
	for i := 2; i > 0; i-- {
		if i > len(components) {
			continue
		}
		split, ok := androidInstallLocationSplits[filepath.Join(components[0:i]...)]
		if ok {
			relStart = split
			break
		}
	}

	if relStart > len(components) {
		relStart = len(components)
	}

	base := filepath.Join(components[:relStart]...)
	rel := filepath.Join(components[relStart:]...)

	return base, rel
}

func getAndroidInstallPath(props *InstallableProps) (string, string, bool) {
	installPath, ok := props.getInstallPath()
	if !ok {
		return "", "", false
	}

	base, rel := splitAndroidPath(installPath)
	return base, rel, true
}
