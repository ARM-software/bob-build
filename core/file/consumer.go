package file

import (
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/google/blueprint"
)

// `Consumer` interface describes a module that is capable of consuming sources for its actions.
// Example of this include:
//   - bob_binary
//   - bob{_static_,_shared_,_}library
//
// This interface can retrieve the required source file for this module only via `GetSrcs`, note that
// each provider needs to be ready by the time these are accessed. This means, `GetSrcs` should be called
// after `ResolveOutFiles` and `processPaths` for each of the dependant modules.
//
// The exception to this is `ResolveOutFiles` which may depend on other dynamic providers, in this case a bottom up
// mutator is used to ensure the downstream dependencies of each module are ready.
type Consumer interface {

	// Returns all sources this module consumes. At this point assumes all providers are ready.
	// Paths will be fully resolved.
	GetFiles(blueprint.BaseModuleContext) Paths

	// Returns a list of targets this consumer directly requires
	GetTargets() []string

	// Returns filepaths for current module only.
	// Context is required for backend information but the accessor should only read current module.
	GetDirectFiles() Paths
}

// Basic common implementation, certain targets may wish to customize this.
func ReferenceGetFilesImpl(ctx blueprint.BaseModuleContext) (srcs Paths) {
	ctx.WalkDeps(
		func(child, parent blueprint.Module) bool {
			isFilegroup := ctx.OtherModuleDependencyTag(child) == tag.FilegroupTag
			_, isConsumer := child.(Consumer)
			_, isProvider := child.(Provider)

			if isFilegroup && isProvider {
				var provided Paths
				child.(Provider).OutFiles().ForEachIf(
					func(fp Path) bool {
						return fp.IsNotType(TypeRsp) && fp.IsNotType(TypeDep)
					},
					func(fp Path) bool {
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
