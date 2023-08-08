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

// TransformSources needs to figure out the output names based on it's inputs.
// Since this cannot be done at `OutSrcs` time due to lack of module context we use a seperate mutator stage.
func ResolveFilesMutator(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(Resolver); ok {
		m.ResolveFiles(ctx)
	}
}
