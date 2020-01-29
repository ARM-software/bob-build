// +build soong

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
package prebuilt

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("prebuilt_data_bob", PrebuiltDataFactory)
}

// Prebuilts going to data partition
type PrebuiltData struct {
	// pretend to be soong prebuilt module
	android.PrebuiltEtc
}

// implemented interfaces check
var _ android.Module = (*PrebuiltData)(nil)
var _ android.AndroidMkEntriesProvider = (*PrebuiltData)(nil)

func PrebuiltDataFactory() android.Module {
	m := &PrebuiltData{}
	// register PrebuiltEtc properties,
	// install path will be relative to data partition root
	android.InitPrebuiltEtcModule(&m.PrebuiltEtc, "")

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibFirst)

	return m
}

func (m *PrebuiltData) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "DATA",
		OutputFile: android.OptionalPathForPath(m.OutputFile()),
		Include:    "$(BUILD_PREBUILT)",
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", m.InstallDirPath().ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", m.OutputFile().Base())
			},
		},
	}}
}

// required to generate ninja rule for copying files onto data partition
func (m *PrebuiltData) InstallInData() bool {
	return true
}
