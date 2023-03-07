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
	"os"
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
	Exclude_directories *bool

	// Error-out if the result `Files` is empty
	Allow_empty *bool

	// Found module sources
	Files []string `blueprint:"mutated"`
}

type moduleGlob struct {
	moduleBase
	Properties struct {
		GlobProps
	}
}

func (m *moduleGlob) shortName() string {
	return m.Name()
}

func (m *moduleGlob) getSourceFiles(ctx blueprint.BaseModuleContext) []string {
	return m.Properties.Files
}

func (m *moduleGlob) getSourceTargets(ctx blueprint.BaseModuleContext) []string {
	return []string{}
}

func (m *moduleGlob) getSourcesResolved(ctx blueprint.BaseModuleContext) []string {
	return m.getSourceFiles(ctx)
}

func (m *moduleGlob) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {

	if len(m.Properties.Srcs) == 0 {
		ctx.PropertyErrorf("srcs", "Missed required property.")
		return
	}

	prefix := filepath.Join(getSourceDir(), ctx.ModuleDir())
	excludes := utils.PrefixDirs(m.Properties.Exclude, prefix)

	addPath := func(p string) {
		dir := filepath.Join(prefix, p)
		isDir, err := ctx.Fs().IsDir(dir)
		if os.IsNotExist(err) || err != nil {
			ctx.ModuleErrorf("glob failed with: %s", err)
		}
		if !(isDir && *m.Properties.Exclude_directories) {
			m.Properties.Files = append(m.Properties.Files, p)
		}
	}

	for _, file := range m.Properties.Srcs {
		if strings.ContainsAny(file, "*?[") {
			// Globs need to be calculated relative to the module
			// directory but not current working directory,
			// thus need to be prefixed with `source_dir/module_dir`
			// (i.e. `getSourceDir() + ctx.ModuleDir()`)
			// so add it to `file`, and remove it afterwards.

			file = filepath.Join(prefix, file)
			matches, err := ctx.GlobWithDeps(file, excludes)

			if err != nil {
				ctx.ModuleErrorf("glob failed with: %s", err)
			}

			for _, match := range matches {
				rel, err := filepath.Rel(prefix, match)
				if err != nil {
					panic(err)
				}
				addPath(rel)
			}
		} else if !utils.Contains(m.Properties.Exclude, file) {
			addPath(file)
		}
	}

	if len(m.Properties.Files) == 0 && !(*m.Properties.Allow_empty) {
		ctx.ModuleErrorf("Glob is empty!")
	}

	for _, s := range append(m.Properties.Srcs, m.Properties.Exclude...) {
		if strings.HasPrefix(filepath.Clean(s), "../") {
			g.getLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	m.Properties.Files = utils.PrefixDirs(m.Properties.Files, ctx.ModuleDir())
}

func (g *moduleGlob) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// `moduleGlob` does not need any generate actions.
	// Only sources should be returned to the modules depending on.
}

func (g moduleGlob) GetProperties() interface{} {
	return g.Properties
}

func globFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	t := true
	module := &moduleGlob{}

	// set `Allow_empty` and `Exclude_directories` to true
	// to match Bazel's `glob`
	module.Properties.GlobProps.Exclude_directories = &t
	module.Properties.GlobProps.Allow_empty = &t

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
