/*
 * Copyright 2023 Arm Limited.
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

package file

import (
	"path"
	"path/filepath"
	"sync"
)

// File factory, stores the runtime configuration for different backends
// This avoids the circular dependancy on the generator backend and reduces
// the data overhead required by the file interface.
type Factory struct {
	buildDir  string
	sourceDir string
}

var singleton *Factory
var lock = &sync.Mutex{}

func FactorySetup(buildDir string, sourceDir string) {
	if singleton == nil {
		lock.Lock()
		defer lock.Unlock()
		if singleton == nil {
			singleton = &Factory{
				buildDir:  buildDir,
				sourceDir: sourceDir,
			}
		}
	}

}

func GetFactory() *Factory {
	if singleton == nil {
		lock.Lock()
		defer lock.Unlock()
		if singleton == nil {
			panic("The File factory has not been configured, please call FactorySetup() before use.")
		}
	}

	return singleton
}

func (factory Factory) New(relativePath string, namespace string, tag Type) Path {
	// TODO: remove generator backend here
	// TODO: add noncompiled dep tag
	switch path.Ext(relativePath) {
	case ".s", ".S":
		tag |= TypeAsm
	case ".c":
		tag |= TypeC
	case ".cc", ".cpp", ".cxx":
		tag |= TypeCpp
	case ".h", ".hpp":
		tag |= TypeHeader
		// TODO: .so .a .o
	}

	var backendPath string
	var scopedPath string
	if (tag & (TypeBinary | TypeExecutable)) != 0 {
		backendPath = factory.buildDir
	} else if (tag & TypeGenerated) != 0 {
		backendPath = filepath.Join(factory.buildDir, "gen")
		scopedPath = namespace
	} else {
		backendPath = factory.sourceDir
		scopedPath = FileNoNameSpace
	}

	return Path{
		backendPath:   backendPath,
		namespacePath: scopedPath,
		relativePath:  relativePath,
		tag:           tag,
	}
}
