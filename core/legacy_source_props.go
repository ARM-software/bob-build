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

// LegacySourceProps defines module properties that are used to identify the
// source files associated with a module. These are used for legacy targets, new
// targets should use `SourceProps` where possible.
type LegacySourceProps struct {
	// The list of source files. Wildcards can be used (but are suboptimal)
	Srcs []string
	// The list of source files that should not be included. Use with care.
	Exclude_srcs []string
	// A list of filegroup modules that provide srcs, these are directly added to Srcs.
	// We do not currently re-use Srcs for this
	Filegroup_srcs []string
}

func (s *LegacySourceProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)

	for _, src := range s.Srcs {
		if strings.HasPrefix(filepath.Clean(src), "../") {
			g.getLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	if len(s.Filegroup_srcs) > 0 {
		g.getLogger().Warn(warnings.DeprecatedFilegroupSrcs, ctx.BlueprintsFile(), ctx.ModuleName())
	}

	srcs := utils.PrefixDirs(utils.Filter(func(s string) bool { return s[0] != ':' }, s.Srcs), prefix)
	srcs = append(srcs, utils.Filter(func(s string) bool { return s[0] == ':' }, s.Srcs)...)

	s.Srcs = srcs
	s.Exclude_srcs = utils.PrefixDirs(s.Exclude_srcs, prefix)
}

// Get a list of sources which are files
func (s *LegacySourceProps) getSourceFiles(ctx blueprint.BaseModuleContext) []string {
	return utils.Filter(func(s string) bool { return s[0] != ':' }, s.Srcs)
}

func (s *LegacySourceProps) getSourceTargets(ctx blueprint.BaseModuleContext) []string {
	return append(utils.StripPrefixAll(utils.Filter(func(s string) bool { return s[0] == ':' }, s.Srcs), ":"), s.Filegroup_srcs...)
}

// Get a list of sources to compile.
//
// The sources are relative to the project directory (i.e. include
// the module directory but not the base source directory), and
// excludes have been handled.
//
// Legacy mode supports globbing.
func (s *LegacySourceProps) getSourcesResolved(ctx blueprint.BaseModuleContext) []string {
	return glob(ctx, append(s.getSourceFiles(ctx), getFileGroupDeps(ctx)...), s.Exclude_srcs)
}
