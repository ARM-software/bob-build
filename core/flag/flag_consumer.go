package flag

import (
	"github.com/google/blueprint"
)

type Consumer interface {
	// Returns flags consumed by this module, listed by itself
	FlagsIn() Flags

	// Returns flags consumed by this module, including any flags provided by its upstream providers.
	FlagsInTransitive(blueprint.BaseModuleContext) Flags
}

// Basic common implementation, certain targets may wish to customize this.
func ReferenceFlagsInTransitive(ctx blueprint.BaseModuleContext) (ret Flags) {

	//There are a few possible ways for flags to be propagated. In legacy targets there are:
	// * Direct dependancies
	// * Indirect dependancies via reexport_libs
	// In new targets there are also:
	// * Transitive exports. These make `reexport_libs` obsolete.

	visited := map[string]bool{}

	ctx.VisitDirectDeps(func(child blueprint.Module) {
		if visited[child.Name()] {
			return
		}
		visited[child.Name()] = true

		if provider, ok := child.(Provider); ok {
			flags := provider.FlagsOut().Filtered(
				func(f Flag) bool {
					return f.MatchesType(TypeExported | TypeTransitive)
				},
			)

			flags.ForEach(func(f Flag) {
				ret = ret.AppendIfUnique(f)
			})
		}
	})

	// Handle transitive flags. Any previously visited module should be skipped as Transitive flags are normally exported as well.
	ctx.WalkDeps(func(child, parent blueprint.Module) bool {
		if visited[child.Name()] {
			return true
		}
		visited[child.Name()] = true
		if provider, ok := child.(Provider); ok {
			flags := provider.FlagsOut().Filtered(
				func(f Flag) bool {
					return f.MatchesType(TypeTransitive)
				},
			)

			flags.ForEach(func(f Flag) {
				ret = ret.AppendIfUnique(f)
			})
		}

		return true
	})

	return
}
