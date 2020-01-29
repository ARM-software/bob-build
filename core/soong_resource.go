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
package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/plugins/prebuilt"

	"android/soong/android"

	"github.com/google/blueprint/proptools"
)

// Properties we need to set in the embedded PrebuiltEtc struct
type prebuiltEtcProperties struct {
	// Source file of this prebuilt.
	Src *string

	// optional subdirectory under which this file is installed into
	Sub_dir *string

	// when set to true, and filename property is not set, the name for the installed file
	// is the same as the file name of the source file.
	Filename_from_src *bool

	// Whether this module is directly installable to one of the partitions. Default: true.
	Installable *bool
}

func (m *resource) soongBuildActions(mctx android.TopDownMutatorContext) {
	if !isEnabled(m) {
		return
	}

	provenanceProps := getProvenanceProps(&m.Properties.AndroidProps)

	installProps := m.getInstallableProps()
	installPath, ok := installProps.getInstallGroupPath()
	if !ok {
		installPath = ""
	} else {
		if installProps.Relative_install_path != nil {
			installPath = filepath.Join(installPath, proptools.String(installProps.Relative_install_path))
		}
	}

	subdir := ""
	factory := (func() android.Module)(nil)
	if strings.HasPrefix(installPath, "data/") {
		subdir = strings.Replace(installPath, "data/", "", 1)
		factory = prebuilt.PrebuiltDataFactory
	} else if strings.HasPrefix(installPath, "etc/") {
		subdir = strings.Replace(installPath, "etc/", "", 1)
		factory = android.PrebuiltEtcFactory
	} else {
		panic(fmt.Errorf("Install path must be prefixed either with 'data' or 'etc' (%s)", installPath))
	}

	// as prebuilt_etc module supports only single src, we have to split into N modules
	for _, src := range m.Properties.getSources(mctx) {
		// prebuilt_etc expects src to not contain a module dir, so we have to strip it here
		base_src := relativeToModuleDir(mctx, []string{src})[0]

		nameProps := nameProps{
			// keep module name unique, remove slashes
			proptools.StringPtr(m.Name() + "__" + strings.Replace(base_src, "/", "_", -1)),
		}

		props := prebuiltEtcProperties{
			Src:               proptools.StringPtr(base_src),
			Sub_dir:           proptools.StringPtr(subdir),
			Filename_from_src: proptools.BoolPtr(true),
			Installable:       proptools.BoolPtr(true),
		}

		// create module and fill all its registered properties with data from prepared structs
		mctx.CreateModule(factory, &nameProps, provenanceProps, &props)
	}
}
