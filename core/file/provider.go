package file

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
