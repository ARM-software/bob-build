package flag

type Provider interface {
	// Returns flags provided by this module to any consumer.
	FlagsOut() Flags
}

type ReExporter interface {
	Provider

	/* This interface is required for targets which reexport libs */
	FlagsOutTargets() []string
}
