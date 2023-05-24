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

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
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

	ResolvedSrcs file.Paths `blueprint:"mutated"` // Glob results.
}

// All interfaces supported by LegacySourceProps
type LegacySourcePropsInterface interface {
	pathProcessor
	FileConsumer
	FileResolver
}

var _ LegacySourcePropsInterface = (*LegacySourceProps)(nil) // impl check

func (s *LegacySourceProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)

	for _, src := range s.Srcs {
		if strings.HasPrefix(filepath.Clean(src), "../") {
			backend.Get().GetLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	if len(s.Filegroup_srcs) > 0 {
		backend.Get().GetLogger().Warn(warnings.DeprecatedFilegroupSrcs, ctx.BlueprintsFile(), ctx.ModuleName())
	}

	srcs := utils.MixedListToFiles(s.Srcs)
	targets := utils.MixedListToBobTargets(s.Srcs)

	s.Srcs = append(utils.PrefixDirs(srcs, prefix), utils.PrefixAll(targets, ":")...)
	s.Exclude_srcs = utils.PrefixDirs(s.Exclude_srcs, prefix)
}

func (s *LegacySourceProps) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	// Since globbing is supported we must call a resolver.
	files := file.Paths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(s.Srcs), s.Exclude_srcs) {
		fp := file.NewPath(match, ctx.ModuleName(), 0)
		files = files.AppendIfUnique(fp)
	}

	s.ResolvedSrcs = files
}

func (s *LegacySourceProps) GetTargets() []string {
	return append(s.Filegroup_srcs, utils.MixedListToBobTargets(s.Srcs)...)
}

func (s *LegacySourceProps) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return s.GetDirectFiles().Merge(ReferenceGetFilesImpl(ctx))
}

func (s *LegacySourceProps) GetDirectFiles() file.Paths {
	return s.ResolvedSrcs
}
