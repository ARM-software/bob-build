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

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

type SourceProps struct {
	// The list of source files and target names for globs/filegroups
	Srcs []string

	ResolvedSrcs file.Paths `blueprint:"mutated"` // Glob results.
}

// Reusable baseline implementation. Each module should match this interface.
var _ pathProcessor = (*SourceProps)(nil)
var _ file.Consumer = (*SourceProps)(nil)

// Helper function to process source paths for Modules using `SourceProps`
func (s *SourceProps) processPaths(ctx blueprint.BaseModuleContext) {

	prefix := projectModuleDir(ctx)

	for _, src := range s.Srcs {
		if strings.HasPrefix(filepath.Clean(src), "../") {
			backend.Get().GetLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	srcs := utils.MixedListToFiles(s.Srcs)
	targets := utils.MixedListToBobTargets(s.Srcs)
	s.Srcs = append(utils.PrefixDirs(srcs, prefix), utils.PrefixAll(targets, ":")...)
}

// Basic common implementation, certain targets will custmize this.
func (s *SourceProps) GetTargets() []string {
	return utils.MixedListToBobTargets(s.Srcs)
}

// Basic common implementation, certain targets may wish to customize this.
func ReferenceGetFilesImpl(ctx blueprint.BaseModuleContext) (srcs file.Paths) {
	ctx.WalkDeps(
		func(child, parent blueprint.Module) bool {
			isFilegroup := ctx.OtherModuleDependencyTag(child) == FilegroupTag
			_, isConsumer := child.(file.Consumer)
			_, isProvider := child.(file.Provider)

			if isFilegroup && isProvider {
				var provided file.Paths
				child.(file.Provider).OutFiles().ForEachIf(
					func(fp file.Path) bool {
						return fp.IsNotType(file.TypeRsp) && fp.IsNotType(file.TypeDep)
					},
					func(fp file.Path) bool {
						provided = append(provided, fp)
						return true
					})
				srcs = srcs.Merge(provided)
			}

			// Only continue if the child is a provider and not a consumer.
			// This means if a consumer eats up downstream providers it should process and output them first.
			return isProvider && !isConsumer
		},
	)

	return
}

func (s *SourceProps) GetDirectFiles() file.Paths {
	return s.ResolvedSrcs
}

func (s *SourceProps) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return s.GetDirectFiles().Merge(ReferenceGetFilesImpl(ctx))
}

// A dynamic source provider is a module which needs to compute the output file names.
//
// `ResolveOutFiles`, is context aware, and runs bottom up in the dep graph. This means however it cannot run
// in parallel, fortunately this is __only__ used for `bob_transform_source`.
//
// `ResolveOutFiles` is context aware specifically because it can depend on other dynamic providers.
type DynamicFileProvider interface {
	file.Provider
	ResolveOutFiles(blueprint.BaseModuleContext)
}

// TransformSources needs to figure out the output names based on it's inputs.
// Since this cannot be done at `OutSrcs` time due to lack of module context we use a seperate mutator stage.
func resolveDynamicFileOutputs(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(DynamicFileProvider); ok {
		m.ResolveOutFiles(ctx)
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
	ResolveFiles(blueprint.BaseModuleContext)
}

func (s *SourceProps) ResolveFiles(ctx blueprint.BaseModuleContext) {
	// Since globbing is supported we must call a resolver.
	files := file.Paths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(s.Srcs), []string{}) {
		fp := file.NewPath(match, ctx.ModuleName(), 0)
		files = files.AppendIfUnique(fp)
	}

	s.ResolvedSrcs = files
}

// TransformSources needs to figure out the output names based on it's inputs.
// Since this cannot be done at `OutSrcs` time due to lack of module context we use a seperate mutator stage.
func resolveFilesMutator(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(FileResolver); ok {
		m.ResolveFiles(ctx)
	}
}
