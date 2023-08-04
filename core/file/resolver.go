package file

import "github.com/google/blueprint"

// `processPaths` needs to run separately to `SourceFileResolver`.
// This is due to how the defaults are resolved and applied, meaning only defaultable properties will be merged.
// The current flow is:
//   - `processPaths` prepends the module subdirectory to source file.
//   - `DefaultApplierMutator` runs, merging source attributes.
//   - `ResolveSrcs` runs, setting up filepaths for distribution.
type Resolver interface {
	// TODO: This may not be neccessary.
	ResolveFiles(blueprint.BaseModuleContext)
}
