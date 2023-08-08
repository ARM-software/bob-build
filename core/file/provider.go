package file

import "github.com/google/blueprint"

// A `Provider` describes a class capable of providing source files,
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
// static sources, and `ResolveOutFiles` for sources which use a dynamic pathing.
//
// `processPaths` should not require context and only operate on the current module.
type Provider interface {
	// Sources to be forwarded to other modules
	// Expected to be called from a context of another module.
	OutFiles() Paths

	// Targets to be forwarded to other modules
	// Expected to be called from a context of another module.
	OutFileTargets() []string
}

// A dynamic source provider is a module which needs to compute the output file names.
//
// `ResolveOutFiles`, is context aware, and runs bottom up in the dep graph. This means however it cannot run
// in parallel, fortunately this is __only__ used for `bob_transform_source`.
//
// `ResolveOutFiles` is context aware specifically because it can depend on other dynamic providers.
type DynamicProvider interface {
	Provider
	ResolveOutFiles(blueprint.BaseModuleContext)
}
