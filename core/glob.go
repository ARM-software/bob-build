package core

import (
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"

	"github.com/google/blueprint"
)

type GlobProps struct {
	// Path patterns that are relative to the current module
	Srcs []string

	// Path patterns that are relative to the current module to exclude from `Srcs`
	Exclude []string

	// Omitted directories from the `Files` result
	Exclude_directories *bool // Currently no supported.

	// Error-out if the result `Files` is empty
	Allow_empty *bool

	// Found module sources
	Files file.Paths `blueprint:"mutated"`
}

type ModuleGlob struct {
	module.ModuleBase
	Properties struct {
		GlobProps
	}
}

// All interfaces supported by moduleGlob
type moduleGlobInterface interface {
	pathProcessor
	FileResolver
	FileProvider
}

var _ moduleGlobInterface = (*ModuleGlob)(nil) // impl check

func (m *ModuleGlob) shortName() string {
	return m.Name()
}

func (m *ModuleGlob) processPaths(ctx blueprint.BaseModuleContext) {
	if len(m.Properties.Srcs) == 0 {
		ctx.PropertyErrorf("srcs", "Missed required property.")
		return
	}

	for _, s := range append(m.Properties.Srcs, m.Properties.Exclude...) {
		if strings.HasPrefix(filepath.Clean(s), "../") {
			backend.Get().GetLogger().Warn(warnings.RelativeUpLinkWarning, ctx.BlueprintsFile(), ctx.ModuleName())
		}
	}

	prefix := ctx.ModuleDir()
	m.Properties.Srcs = utils.PrefixDirs(m.Properties.Srcs, prefix)
	m.Properties.Exclude = utils.PrefixDirs(m.Properties.Exclude, prefix)
}

func (m *ModuleGlob) ResolveFiles(ctx blueprint.BaseModuleContext) {
	matches := glob(ctx, m.Properties.Srcs, m.Properties.Exclude)
	files := file.Paths{}

	for _, match := range matches {
		fp := file.NewPath(match, ctx.ModuleName(), 0)
		files = files.AppendIfUnique(fp)
	}

	if len(files) == 0 && !(*m.Properties.Allow_empty) {
		ctx.ModuleErrorf("Glob is empty!")
	}

	m.Properties.Files = files

}

func (m *ModuleGlob) OutFiles() file.Paths {
	return m.Properties.Files
}

func (m *ModuleGlob) OutFileTargets() (tgts []string) {
	// does not forward any of it's source providers.
	return
}

func (m *ModuleGlob) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// `moduleGlob` does not need any generate actions.
	// Only sources should be returned to the modules depending on.
}

func (m ModuleGlob) GetProperties() interface{} {
	return m.Properties
}

func globFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	t := true
	module := &ModuleGlob{}

	// set `Allow_empty` and `Exclude_directories` to true
	// to match Bazel's `glob`
	module.Properties.GlobProps.Exclude_directories = &t
	module.Properties.GlobProps.Allow_empty = &t

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
