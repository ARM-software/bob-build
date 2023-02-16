//go:build soong
// +build soong

/*
 * Copyright 2020-2023 Arm Limited.
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
package prebuilt

import (
	"android/soong/android"
	"android/soong/etc"

	"github.com/ARM-software/bob-build/internal/soong_compat"
)

func init() {
	android.RegisterModuleType("prebuilt_data_bob", PrebuiltDataFactory)
	android.RegisterModuleType("prebuilt_testcase_bob", PrebuiltTestcaseFactory)
}

// Prebuilts going to data partition
type PrebuiltData struct {
	// pretend to be soong prebuilt module
	etc.PrebuiltEtc
}

// implemented interfaces check
var _ android.Module = (*PrebuiltData)(nil)
var _ android.AndroidMkEntriesProvider = (*PrebuiltData)(nil)

func PrebuiltDataFactory() android.Module {
	m := &PrebuiltData{}
	// register PrebuiltEtc properties,
	// install path will be relative to data partition root
	etc.InitPrebuiltEtcModule(&m.PrebuiltEtc, "")

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibFirst)

	return m
}

func (m *PrebuiltData) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "DATA",
		OutputFile: android.OptionalPathForPath(m.OutputFile()),
		Include:    "$(BUILD_PREBUILT)",
		ExtraEntries: soong_compat.ConvertAndroidMkExtraEntriesFunc(
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", m.InstallDirPath().ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", m.OutputFile().Base())
			},
		),
	}}
}

// required to generate ninja rule for copying files onto data partition
func (m *PrebuiltData) InstallInData() bool {
	return true
}

// Prebuilts going to data partition
type PrebuiltTestcase struct {
	// pretend to be soong prebuilt module
	etc.PrebuiltEtc
}

// implemented interfaces check
var _ android.Module = (*PrebuiltTestcase)(nil)
var _ android.AndroidMkEntriesProvider = (*PrebuiltTestcase)(nil)

func PrebuiltTestcaseFactory() android.Module {
	m := &PrebuiltTestcase{}
	// register PrebuiltEtc properties,
	// install path will be relative to the `testcases` directory
	etc.InitPrebuiltEtcModule(&m.PrebuiltEtc, "")

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibFirst)

	return m
}

func (m *PrebuiltTestcase) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "DATA",
		OutputFile: android.OptionalPathForPath(m.OutputFile()),
		Include:    "$(BUILD_PREBUILT)",
		ExtraEntries: soong_compat.ConvertAndroidMkExtraEntriesFunc(
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", m.InstallDirPath().ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", m.OutputFile().Base())
			},
		),
	}}
}

// required to generate ninja rule for copying files into the `testcases` directory
func (m *PrebuiltTestcase) InstallInTestcases() bool {
	return true
}
