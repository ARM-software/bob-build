package core

import (
	"strings"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type StrictGenerateProps struct {
	// See https://ci.android.com/builds/submitted/8928481/linux/latest/view/soong_build.html
	Name                string
	Srcs                []string // TODO: This module should probalby make use of LegacySourceProps
	Exclude_srcs        []string
	Cmd                 *string
	Depfile             *bool
	Export_include_dirs []string
	Tool_files          []string
	Tools               []string

	ResolvedSrcs file.Paths `blueprint:"mutated"` // Glob results.
}

type StrictGeneratePropsInterface interface {
	pathProcessor
	file.Consumer
	file.Resolver
}

var _ StrictGeneratePropsInterface = (*StrictGenerateProps)(nil) // impl check

func (ag *StrictGenerateProps) processPaths(ctx blueprint.BaseModuleContext) {

	prefix := projectModuleDir(ctx)
	// We don't want to process module dependencies as paths, we must filter them out first.

	srcs := utils.MixedListToFiles(ag.Srcs)
	targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Srcs), ":")

	ag.Srcs = append(utils.PrefixDirs(srcs, prefix), targets...)
	ag.Exclude_srcs = utils.PrefixDirs(ag.Exclude_srcs, prefix)

	ag.validateCmd(ctx)

	// When we specify a specific tag, its location will be incorrect as we move everything into a top level bp,
	// we must fix this by iterating through the command.
	matches := locationRegex.FindAllStringSubmatch(*ag.Cmd, -1)
	for _, v := range matches {
		tag := v[1]
		if tag[0] == ':' {
			continue
		}

		// do not prefix paths for `Tools` which are host binary modules
		if utils.Contains(ag.Tool_files, tag) {
			newTag := utils.PrefixDirs([]string{tag}, prefix)[0]
			// Replacing with space allows us to not replace the same basename more than once if it appears
			// multiple times.
			newCmd := strings.Replace(*ag.Cmd, " "+tag, " "+newTag, -1)
			ag.Cmd = &newCmd
		}
	}

	tool_files_targets := utils.PrefixAll(utils.MixedListToBobTargets(ag.Tool_files), ":")
	ag.Tool_files = utils.PrefixDirs(utils.MixedListToFiles(ag.Tool_files), prefix)
	ag.Tool_files = append(ag.Tool_files, tool_files_targets...)
}

func (ag *StrictGenerateProps) validateCmd(ctx blueprint.BaseModuleContext) {

	// for variables only curly brackets are allowed
	matches := variableRegex.FindAllStringSubmatch(*ag.Cmd, -1)

	for _, v := range matches {
		ctx.ModuleErrorf("Only curly brackets are allowed in `cmd`. Use: '${%s}'", v[1])
	}

	// Check default tool
	if strings.Contains(*ag.Cmd, "${location}") {
		if len(ag.Tools) > 0 && len(ag.Tool_files) > 0 {
			ctx.ModuleErrorf("You cannot have default $(location) specified in `cmd` if setting both `tool_files` and `tools`.")
		}
	}
}

func (ag *StrictGenerateProps) ResolveFiles(ctx blueprint.BaseModuleContext) {
	// Since globbing is supported we must call a resolver.
	files := file.Paths{}

	for _, match := range glob(ctx, utils.MixedListToFiles(ag.Srcs), ag.Exclude_srcs) {
		fp := file.NewPath(match, ctx.ModuleName(), file.TypeUnset)
		files = files.AppendIfUnique(fp)
	}

	ag.ResolvedSrcs = files
}

func (ag *StrictGenerateProps) GetTargets() []string {
	return utils.MixedListToBobTargets(ag.Srcs)
}

func (ag *StrictGenerateProps) GetDirectFiles() file.Paths {
	return ag.ResolvedSrcs
}

func (ag *StrictGenerateProps) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return ag.GetDirectFiles().Merge(file.ReferenceGetFilesImpl(ctx))
}
