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
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

type GlobProps struct {
	// Path patterns that are relative to the current module
	Srcs []string

	// Path patterns that are relative to the current module to exclude from `Srcs`
	Exclude []string

	// Omitted directories from the `Files` result
	Exclude_directories *bool // Currently no supported.

	// Error-out if the result `Files` is empty
	Allow_empty *bool

	// Found module sources
	Files FilePaths `blueprint:"mutated"`
}

type ModuleGlob struct {
	moduleBase
	Properties struct {
		GlobProps
	}
}

// All interfaces supported by moduleGlob
type moduleGlobInterface interface {
	pathProcessor
	FileResolver
	SourceFileProvider
}

var _ moduleGlobInterface = (*ModuleGlob)(nil) // impl check

func (m *ModuleGlob) shortName() string {
	return m.Name()
}

func (m *ModuleGlob) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	if len(m.Properties.Srcs) == 0 {
		ctx.PropertyErrorf("srcs", "Missed required property.")
		return
	}

	for _, s := range append(m.Properties.Srcs, m.Properties.Exclude...) {
		if strings.HasPrefix(filepath.Clean(s), "../") {
			g.getLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	prefix := ctx.ModuleDir()
	m.Properties.Srcs = utils.PrefixDirs(m.Properties.Srcs, prefix)
	m.Properties.Exclude = utils.PrefixDirs(m.Properties.Exclude, prefix)
}

func (m *ModuleGlob) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	matches := glob(ctx, m.Properties.Srcs, m.Properties.Exclude)
	files := FilePaths{}

	for _, match := range matches {
		fp := newSourceFilePath(match, ctx, g)
		files = files.AppendIfUnique(fp)
	}

	if len(files) == 0 && !(*m.Properties.Allow_empty) {
		ctx.ModuleErrorf("Glob is empty!")
	}

	m.Properties.Files = files

}

func (m *ModuleGlob) OutSrcs() FilePaths {
	return m.Properties.Files
}

func (m *ModuleGlob) OutSrcTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleGlob) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// `moduleGlob` does not need any generate actions.
	// Only sources should be returned to the modules depending on.
}

func (m ModuleGlob) GetProperties() interface{} {
	return m.Properties
}

func globFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	t := true
	module := &ModuleGlob{}

	// set `Allow_empty` and `Exclude_directories` to true
	// to match Bazel's `glob`
	module.Properties.GlobProps.Exclude_directories = &t
	module.Properties.GlobProps.Allow_empty = &t

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
