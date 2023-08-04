package file

import (
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
