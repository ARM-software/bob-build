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
	"fmt"
	"strings"

	"github.com/google/blueprint"
)

func (g *androidBpGenerator) resourceActions(r *resource, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(r) {
		return
	}

	installBase, installRel, _ := getAndroidInstallPath(r.getInstallableProps())

	var modType string
	if strings.HasPrefix(installBase+"/", "data/") {
		modType = "prebuilt_data_bob"
	} else if strings.HasPrefix(installBase+"/", "etc/") {
		modType = "prebuilt_etc"
	} else if strings.HasPrefix(installBase+"/", "firmware/") {
		modType = "prebuilt_firmware"
	} else {
		panic(fmt.Errorf("Could not detect partition for install path '%s'", installBase))
	}

	// as prebuilt_etc module supports only single src, we have to split into N modules
	for _, src := range r.Properties.getSources(mctx) {
		// keep module name unique, remove slashes
		modName := r.shortName() + "__" + strings.Replace(src, "/", "_", -1)

		m, err := AndroidBpFile().NewModule(modType, modName)
		if err != nil {
			panic(err.Error())
		}

		addProvenanceProps(m, r.Properties.AndroidProps)

		// add prebuilt_etc properties
		m.AddString("src", src)
		m.AddString("sub_dir", installRel)
		m.AddBool("filename_from_src", true)
		m.AddBool("installable", true)
	}
}
