package core

import (
	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

// CommonProps defines a set of properties which are common
// for multiple module types.
type CommonProps struct {
	LegacySourceProps
	IncludeDirsProps
	InstallableProps
	EnableableProps
	AndroidProps
	AliasableProps

	// Flags used for C compilation
	Cflags []string
}

func (c *CommonProps) processPaths(ctx blueprint.BaseModuleContext) {
	prefix := projectModuleDir(ctx)

	c.LegacySourceProps.processPaths(ctx)
	c.InstallableProps.processPaths(ctx)
	c.IncludeDirsProps.Local_include_dirs = utils.PrefixDirs(c.IncludeDirsProps.Local_include_dirs, prefix)

	// TODO: This should be done in a dedicated mutator for prop checks.
	if c.AndroidProps.Owner != nil {
		backend.Get().GetLogger().Warn(warnings.DeprecatedOwnerProp, ctx.BlueprintsFile(), ctx.ModuleName())
	}
}
