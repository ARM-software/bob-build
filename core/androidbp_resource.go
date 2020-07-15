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
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/bpwriter"
)

func writeDataResourceModule(m bpwriter.Module, src, installRel string) {
	// add prebuilt_etc properties
	m.AddString("src", src)
	m.AddString("sub_dir", installRel)
	m.AddBool("filename_from_src", true)
	m.AddBool("installable", true)
}

func writeCodeResourceModule(m bpwriter.Module, src, installRel string) {
	m.AddStringList("srcs", []string{src})
	m.AddString("stem", filepath.Base(src))
	m.AddString("relative_install_path", installRel)
}

func (g *androidBpGenerator) resourceActions(r *resource, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(r) {
		return
	}

	installBase, installRel, _ := getSoongInstallPath(r.getInstallableProps())

	var modType string
	// Soong has two types of backend modules; "data" ones, for places like
	// /etc, and "code" ones, for locations like /bin. Write different sets
	// of properties depending on which one is required.
	var write func(bpwriter.Module, string, string)

	if installBase == "data" {
		modType = "prebuilt_data_bob"
		write = writeDataResourceModule
	} else if installBase == "etc" {
		modType = "prebuilt_etc"
		write = writeDataResourceModule
	} else if installBase == "firmware" {
		modType = "prebuilt_firmware"
		write = writeDataResourceModule
	} else if installBase == "bin" {
		modType = "cc_prebuilt_binary"
		write = writeCodeResourceModule
	} else if installBase == "tests" {
		// Eventually we want to install in testcases,
		// But we can't put binaries there yet.
		// So place resources in /data/nativetest to align with cc_test.
		//modType = "prebuilt_testcase_bob"
		modType = "prebuilt_data_bob"
		if r.Properties.isProprietary() {
			// Vendor modules need an additional path element to match cc_test
			installRel = filepath.Join("nativetest", "vendor", installRel)
		} else {
			installRel = filepath.Join("nativetest", installRel)
		}
		write = writeDataResourceModule
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

		write(m, src, installRel)
	}
}
