// +build soong

/*
 * Copyright 2019-2020 Arm Limited.
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
	"path/filepath"

	"android/soong/android"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/plugins/kernelmodule"
)

func (m *kernelModule) soongBuildActions(mctx android.TopDownMutatorContext) {
	nameProps := nameProps{
		proptools.StringPtr(m.Name()),
	}

	provenanceProps := getProvenanceProps(&m.Properties.Build.BuildProps.AndroidProps)

	installProps := m.getInstallableProps()
	installPath, ok := installProps.getInstallGroupPath()
	if !ok {
		installPath = ""
	} else {
		if installProps.Relative_install_path != nil {
			installPath = filepath.Join(installPath, proptools.String(installProps.Relative_install_path))
		}
	}

	props := kernelmodule.KernelModuleProps{
		Stem: m.outputName(),
		Args: kernelmodule.KbuildArgs(m.generateKbuildArgs(mctx)),
		// as Srcs are already prefixed with module dir, and soong plugin expects it to be relative to local dir,
		// we have to strip it here
		Srcs:          relativeToModuleDir(mctx, m.Properties.getSources(mctx)),
		Default:       isBuiltByDefault(m),
		Extra_Symbols: m.Properties.Extra_symbols,
		Install_Path:  installPath,
	}

	// create module and fill all its registered properties with data from prepared structs
	mctx.CreateModule(kernelmodule.KernelModuleFactory, &nameProps, provenanceProps, &props)
}
