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

// This module defines interfaces required for consuming and providing sources.
// A module can be a provider, consumer or both. See docs on each of the interfaces
// for details.
//
// filegroupA <- filegroupB <- filegroupC
// A target including filegroupA should see all it's children targets

// In the following case however, filegroupF should only see the result of generateSourceA and transformSourceA
// filegroupF <--- generateSourceA <- generateSourceB
//              |
//               - transformSourceA <- filegroupA

// In this example filegroup is a provider but __not__ a consumer.
// generateSource is both
// transformSource is both
// A provider only interface forwards it's downstream deps

// In the case of generate lib this is even worse:
// binary <---- generateLib <--- filegroupA
//                            |
//                             - transformSourceA <- filegroupF
// In this case generateLib can provide generated headers to binary
// In that sense it should now forward any of it's downstream source providers.

package core

import (
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

type SourceProps struct {
	// The list of source files and target names for globs/filegroups
	Srcs []string

	ResolvedSrcs FilePaths `blueprint:"mutated"` // Glob results.
}

// Reusable baseline implementation. Each module should match this interface.
var _ pathProcessor = (*SourceProps)(nil)
var _ SourceFileConsumer = (*SourceProps)(nil)

// Helper function to process source paths for Modules using `SourceProps`
func (s *SourceProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {

	prefix := projectModuleDir(ctx)

	for _, src := range s.Srcs {
		if strings.HasPrefix(filepath.Clean(src), "../") {
			g.getLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	srcs := utils.MixedListToFiles(s.Srcs)
	targets := utils.MixedListToBobTargets(s.Srcs)
	s.Srcs = append(utils.PrefixDirs(srcs, prefix), utils.PrefixAll(targets, ":")...)
}

// A `SourceFileProvider` descibes a class capable of providing source files,
// either dynamically generated or by reference to other modules. For example:
//   - bob_glob
//   - bob_filegroup
//   - bob_{generate,transform}_source
//   - bob_genrule
//
// A provider interface outputs it's source files for other modules, it can also optionally forward it's source targets
// as is the case for `bob_filegroup`.
//
// `OutSrcs` is not context aware, this is because it is called from a context aware visitor (`GetSrcs`).
// This means its output needs to be resolved prior to this call. This is typically done via `processPaths` for
// static sources, and `ResolveOutSrcs` for sources which use a dynamic pathing.
//
// `processPaths` should not require context and only operate on the current module.
type SourceFileProvider interface {
	// Sources to be forwarded to other modules
	// Expected to be called from a context of another module.
	OutSrcs() FilePaths

	// Targets to be forwarded to other modules
	// Expected to be called from a context of another module.
	OutSrcTargets() []string
}

// TODO: Remove the need for the Implicit file handling.
// Instead add a different filepath type or tag to handle this.
type ImplicitFileProvider interface {
	SourceFileProvider

	// Sources to be forwarded to other modules
	OutImplicits() FilePaths
}

// `SourceFileConsumer` interface describes a module that is capable of consuming sources for its actions.
// Example of this include:
//   - bob_binary
//   - bob{_static_,_shared_,_}library
//
// This interface can retrieve the required source file for this module only via `GetSrcs`, note that
// each provider needs to be ready by the time these are accessed. This means, `GetSrcs` should be called
// after `ResolveOutSrcs` and `processPaths` for each of the dependant modules.
//
// The exception to this is `ResolveOutSrcs` which may depend on other dynamic providers, in this case a bottom up
// mutator is used to ensure the downstream dependencies of each module are ready.
type SourceFileConsumer interface {

	// Returns all sources this module consumes. At this point assumes all providers are ready.
	// Paths will be fully resolved.
	GetSrcs(blueprint.BaseModuleContext) FilePaths

	// Returns a list of targets this consumer directly requires
	GetSrcTargets() []string

	// Returns filepaths for current module only.
	// Context is required for backend information but the accessor should only read current module.
	GetDirectSrcs() FilePaths
}

type ImplicitFileConsumer interface {
	SourceFileConsumer

	GetImplicits(blueprint.BaseModuleContext) FilePaths
}

// Basic common implementation, certain targets will custmize this.
func (s *SourceProps) GetSrcTargets() []string {
	return utils.MixedListToBobTargets(s.Srcs)
}

// Basic common implementation, certain targets may wish to customize this.
func ReferenceGetSrcsImpl(ctx blueprint.BaseModuleContext) (srcs FilePaths) {
	ctx.WalkDeps(
		func(child, parent blueprint.Module) bool {
			isFilegroup := ctx.OtherModuleDependencyTag(child) == filegroupTag
			_, isConsumer := child.(SourceFileConsumer)
			_, isProvider := child.(SourceFileProvider)

			if isFilegroup && isProvider {
				provided := child.(SourceFileProvider).OutSrcs()
				srcs = srcs.Merge(provided)
			}

			// Only continue if the child is a provider and not a consumer.
			// This means if a consumer eats up downstream providers it should process and output them first.
			return isProvider && !isConsumer
		},
	)

	return
}

func ReferenceGetImplicitsImpl(ctx blueprint.BaseModuleContext) (implicits FilePaths) {
	ctx.WalkDeps(
		func(child, parent blueprint.Module) bool {
			isFilegroup := ctx.OtherModuleDependencyTag(child) == filegroupTag
			_, isConsumer := child.(ImplicitFileConsumer)
			_, isProvider := child.(ImplicitFileProvider)

			if isFilegroup && isProvider {
				provided := child.(ImplicitFileProvider).OutImplicits()
				implicits = implicits.Merge(provided)
			}

			// Only continue if the child is a provider and not a consumer.
			// This means if a consumer eats up downstream providers it should process and output them first.
			return isProvider && !isConsumer
		},
	)

	return
}

func (s *SourceProps) GetDirectSrcs() FilePaths {
	return s.ResolvedSrcs
}

func (s *SourceProps) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	return s.GetDirectSrcs().Merge(ReferenceGetSrcsImpl(ctx))
}

// A dynamic source provider is a module which needs to compute the output file names.
//
// `ResolveOutSrcs`, is context aware, and runs bottom up in the dep graph. This means however it cannot run
// in parallel, fortunately this is __only__ used for `bob_transform_source`.
//
// `ResolveOutSrcs` is context aware specifically because it can depend on other dynamic providers.
type DynamicSourceFileProvider interface {
	SourceFileProvider
	ResolveOutSrcs(blueprint.BaseModuleContext)
}

// TransformSources needs to figure out the output names based on it's inputs.
// Since this cannot be done at `OutSrcs` time due to lack of module context we use a seperate mutator stage.
func resolveDynamicSrcOutputs(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(DynamicSourceFileProvider); ok {
		m.ResolveOutSrcs(ctx)
	}
}

// `processPaths` needs to run seperately to `SourceFileResolver`.
// This is due to how the defaults are resolved and applied, meaning only defaultable properties will be merged.
// The current flow is:
//   - `processPaths` prepends the module subdirectory to source file.
//   - `DefaultApplierMutator` runs, merging source attributes.
//   - `ResolveSrcs` runs, setting up filepaths for distribution.
type FileResolver interface {
	// TODO: This may not be neccessary.
	ResolveFiles(blueprint.BaseModuleContext, generatorBackend)
}

func (s *SourceProps) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	// Since globbing is supported we must call a resolver.
	files := FilePaths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(s.Srcs), []string{}) {
		fp := newSourceFilePath(match, ctx, g)
		files = files.AppendIfUnique(fp)
	}

	s.ResolvedSrcs = files
}

// TransformSources needs to figure out the output names based on it's inputs.
// Since this cannot be done at `OutSrcs` time due to lack of module context we use a seperate mutator stage.
func resolveFilesMutator(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(FileResolver); ok {
		m.ResolveFiles(ctx, getBackend(ctx))
	}
}
