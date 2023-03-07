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

package core

import (
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type sharedLibrary struct {
	library
	fileNameExtension string
}

const (
	tocExt = ".toc"
)

// sharedLibrary supports:
// * producing output using the linker
// * producing a shared library
// * stripping symbols from output
var _ linkableModule = (*sharedLibrary)(nil)
var _ sharedLibProducer = (*sharedLibrary)(nil)
var _ stripable = (*sharedLibrary)(nil)

func (l *sharedLibrary) getLinkName() string {
	return l.outputName() + l.fileNameExtension
}

func (l *sharedLibrary) getSoname() string {
	name := l.getLinkName()
	if l.library.Properties.Library_version != "" {
		var v = strings.Split(l.library.Properties.Library_version, ".")
		name += "." + v[0]
	}
	return name
}

func (l *sharedLibrary) getRealName() string {
	name := l.getLinkName()
	if l.library.Properties.Library_version != "" {
		name += "." + l.library.Properties.Library_version
	}
	return name
}

func (l *sharedLibrary) strip() bool {
	return l.Properties.Strip != nil && *l.Properties.Strip
}

func (l *sharedLibrary) librarySymlinks(ctx blueprint.ModuleContext) map[string]string {
	symlinks := map[string]string{}

	if l.library.Properties.Library_version != "" {
		// To build you need a symlink from the link name and soname.
		// At runtime only the soname symlink is required.
		soname := l.getSoname()
		realName := l.getRealName()
		if soname == realName {
			utils.Die("module %s has invalid library_version '%s'",
				l.Name(),
				l.library.Properties.Library_version)
		}
		symlinks[l.getLinkName()] = soname
		symlinks[soname] = realName
	}

	return symlinks
}

func (l *sharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(l) {
		getBackend(ctx).sharedActions(l, ctx)
	}
}

func (l *sharedLibrary) outputFileName() string {
	// Since we link against libraries using the library flag style,
	// -lmod, return the name of the link library here rather than the
	// real, versioned library.
	return l.getLinkName()
}

func (l *sharedLibrary) getTocName() string {
	return l.getRealName() + tocExt
}

func (l sharedLibrary) GetProperties() interface{} {
	return l.library.Properties
}

func sharedLibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &sharedLibrary{}
	if config.Properties.GetBool("osx") {
		module.fileNameExtension = ".dylib"
	} else {
		module.fileNameExtension = ".so"
	}
	return module.LibraryFactory(config, module)
}
