/*
 * Copyright 2020, 2022 Arm Limited.
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

// Android utilities

import (
	"path/filepath"
	"strings"

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
	"$(TARGET_OUT)":         3,
	"$(TARGET_OUT_PRODUCT)": 2,
	"$(TARGET_OUT_SYSTEM)":  2,
	"$(TARGET_OUT_VENDOR)":  2,

	// Android.mk-specific build dir, which a subdirectory per module type.
	"$(TARGET_OUT_GEN)": 2,

	// Filetype-specific Android.mk variables already include the `lib` or `bin` part.
	"bin":                                       1,
	"lib":                                       1,
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

	// /etc can be used on its own (prebuilt_etc) or contain subdir
	// `firmware` which need to be part of the base path
	"etc":                                1,
	"$(TARGET_OUT_DATA_ETC)":             1,
	"$(TARGET_OUT_ETC)":                  1,
	"$(TARGET_OUT_OEM_ETC)":              1,
	"$(TARGET_OUT_PRODUCT_ETC)":          1,
	"$(TARGET_OUT_PRODUCT_SERVICES_ETC)": 1,
	"$(TARGET_OUT_VENDOR_ETC)":           1,

	// Catch etc/firmware in the base path when
	// $(TARGET_OUT_ETC)/firmware is used
	"firmware":                                    1,
	"etc/firmware":                                2,
	"$(TARGET_OUT_DATA_ETC)/firmware":             2,
	"$(TARGET_OUT_ETC)/firmware":                  2,
	"$(TARGET_OUT_OEM_ETC)/firmware":              2,
	"$(TARGET_OUT_PRODUCT_ETC)/firmware":          2,
	"$(TARGET_OUT_PRODUCT_SERVICES_ETC)/firmware": 2,
	"$(TARGET_OUT_VENDOR_ETC)/firmware":           2,

	// /data isn't quite so structured, so put most components in the relative_install_path.
	// Note that $(TARGET_OUT_DATA_EXECUTABLES) etc actually maps to /system, so is handled
	// the same as the other filetype-specific stuff - this just catches anything else.
	"data":               1,
	"$(TARGET_OUT_DATA)": 1,

	// Catch data/nativetest in the base path when
	// $(TARGET_OUT_DATA)/nativetest is used
	"data/nativetest":                2,
	"$(TARGET_OUT_DATA)/nativetest":  2,
	"$(TARGET_OUT_DATA_NATIVE_TEST)": 1,

	// /testcases is unstructured
	"tests":                   1,
	"$(TARGET_OUT_TESTCASES)": 1,
}

func findAndroidInstallLocationSplit(components []string) (int, bool) {
	// Try longer sections of path first to avoid incorrect matches on common prefixes
	for i := 2; i > 0; i-- {
		if i > len(components) {
			continue
		}
		split, ok := androidInstallLocationSplits[filepath.Join(components[0:i]...)]
		if ok {
			return split, true
		}
	}
	// If no match, the whole path is the "base" section.
	return len(components), false
}

func splitAndroidPath(path string) (string, string) {
	components := utils.SplitPath(path)

	relStart, _ := findAndroidInstallLocationSplit(components)

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

// Map of Android.mk variables and the equivalent androidbp backend
// locations. Only locations that it is possible to install to are
// included, so this is a subset of androidInstallLocationSplits.
//
// The identifiers understood by the androidbp backend are bin, lib,
// etc, firmware, data and tests. vendor and system will be inferred
// from the owner property.
var androidMkInstallLocationTranslations = map[string]string{
	"TARGET_OUT":                         "",
	"TARGET_OUT_DATA":                    "data",
	"TARGET_OUT_ETC":                     "etc",
	"TARGET_OUT_EXECUTABLES":             "bin",
	"TARGET_OUT_SHARED_LIBRARIES":        "lib",
	"TARGET_OUT_SYSTEM":                  "",
	"TARGET_OUT_TESTCASES":               "tests",
	"TARGET_OUT_VENDOR":                  "",
	"TARGET_OUT_VENDOR_ETC":              "etc",
	"TARGET_OUT_VENDOR_EXECUTABLES":      "bin",
	"TARGET_OUT_VENDOR_SHARED_LIBRARIES": "lib",
	"TARGET_OUT_DATA_NATIVE_TEST":        "tests",
}

func expandAndroidMkInstallVars(path string) string {
	// Only the first component of a path can be an Android.mk variable
	components := utils.SplitPath(path)

	if len(components) == 0 {
		return path
	}

	varName := strings.TrimSuffix(strings.TrimPrefix(components[0], "$("), ")")
	if len(varName) != len(components[0])-3 {
		// Not all parts were stripped, so this isn't a variable expansion
		return path
	}
	soongPath, ok := androidMkInstallLocationTranslations[varName]
	if !ok {
		return path
	}
	components[0] = soongPath
	return filepath.Join(components...)
}

// After translating make variables like TARGET_OUT, TARGET_OUT_ETC,
// TARGET_OUT_DATA, we may still have multiple path elements. Map
// these to the right androidbp backend location.
var basePathTranslations = map[string]string{
	"etc/firmware":    "firmware",
	"data/nativetest": "tests",
}

func getSoongInstallPath(props *InstallableProps) (string, string, bool) {
	installPath, ok := props.getInstallPath()
	if !ok {
		return "", "", false
	}

	installPath = expandAndroidMkInstallVars(installPath)

	base, rel := splitAndroidPath(installPath)

	base2, ok := basePathTranslations[base]
	if ok {
		base = base2
	}

	return base, rel, true
}

// Identifies if a module links to a generated library. Generated
// libraries only support a single architecture
func linksToGeneratedLibrary(ctx blueprint.ModuleContext) bool {
	seenGeneratedLib := false
	ctx.WalkDeps(func(dep, parent blueprint.Module) bool {
		// Only consider dependencies that get linked
		tag := ctx.OtherModuleDependencyTag(dep)
		if tag == staticDepTag ||
			tag == sharedDepTag ||
			tag == wholeStaticDepTag {
			_, staticLib := dep.(*generateStaticLibrary)
			_, sharedLib := dep.(*generateSharedLibrary)
			if sharedLib || staticLib {
				// We depend on a generated library
				seenGeneratedLib = true
				// No need to continue walking
				return false
			}
			// Keep walking this part of the tree
			return true
		}
		return false
	})
	return seenGeneratedLib
}
