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
			GetLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
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

func (s *SourceProps) GetDirectFiles() file.Paths {
	return s.ResolvedSrcs
}

func (s *SourceProps) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return s.GetDirectFiles().Merge(file.ReferenceGetFilesImpl(ctx))
}

// TransformSources needs to figure out the output names based on it's inputs.
// Since this cannot be done at `OutSrcs` time due to lack of module context we use a seperate mutator stage.
func resolveDynamicFileOutputs(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(file.DynamicProvider); ok {
		m.ResolveOutFiles(ctx)
	}
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
