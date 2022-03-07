// +build soong

/*
 * Copyright 2021-2022 Arm Limited.
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

package soong_compat

import (
	"android/soong/android"
)

// This definition is compatible with Soong SHAs after `aa2555387 Add ctx to
// AndroidMkExtraEntriesFunc`
func ConvertAndroidMkExtraEntriesFunc(f AndroidMkExtraEntriesFunc) []android.AndroidMkExtraEntriesFunc {
	return []android.AndroidMkExtraEntriesFunc{
		func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
			f(entries)
		},
	}
}

func SoongSupportsMkInstallTargets() bool {
	return false
}
