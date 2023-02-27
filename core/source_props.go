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

type SourceProps struct {
	Srcs []string // The list of source files and target names for globs/filegroups
	// generated sources are not yet supported.
}

// Source interface to be implemented for any target that supports sources and filegroups/globs.
type sourceInterface interface {
	getSourceFiles(ctx blueprint.BaseModuleContext) []string     // Returns files listed in `srcs`
	getSourceTargets(ctx blueprint.BaseModuleContext) []string   // Returns targets listed in `srcs` as valid module names
	getSourcesResolved(ctx blueprint.BaseModuleContext) []string // Return resolved list of source files for given module
}

// Helper function to process source paths for Modules using `SourceProps`
func (s *SourceProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)

	for _, src := range s.Srcs {
		if strings.HasPrefix(filepath.Clean(src), "../") {
			g.getLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	srcs := utils.PrefixDirs(s.getSourceFiles(ctx), prefix)
	srcs = append(srcs, utils.Filter(func(s string) bool { return s[0] == ':' }, s.Srcs)...)

	s.Srcs = srcs
}

func (s *SourceProps) getSourceTargets(ctx blueprint.BaseModuleContext) []string {
	return utils.StripPrefixAll(utils.Filter(func(s string) bool { return s[0] == ':' }, s.Srcs), ":")
}

func (s *SourceProps) getSourceFiles(ctx blueprint.BaseModuleContext) []string {
	return utils.Filter(func(s string) bool { return s[0] != ':' }, s.Srcs)
}

func (s *SourceProps) getSourcesResolved(ctx blueprint.BaseModuleContext) []string {
	return append(s.getSourceFiles(ctx), getFileGroupDeps(ctx)...)
}
