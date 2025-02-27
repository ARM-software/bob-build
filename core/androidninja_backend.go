package core

import (
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type androidNinjaGenerator struct {
}

// aliasActions implements generatorBackend.
func (*androidNinjaGenerator) aliasActions(m *ModuleAlias, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// binaryActions implements generatorBackend.
func (*androidNinjaGenerator) binaryActions(m *ModuleBinary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// executableTestActions implements generatorBackend.
func (*androidNinjaGenerator) executableTestActions(m *ModuleTest, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// filegroupActions implements generatorBackend.
func (*androidNinjaGenerator) filegroupActions(m *ModuleFilegroup, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genBinaryActions implements generatorBackend.
func (*androidNinjaGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genSharedActions implements generatorBackend.
func (*androidNinjaGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// genStaticActions implements generatorBackend.
func (*androidNinjaGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

func (g *androidNinjaGenerator) buildRules(r *ruleContext, ctx blueprint.ModuleContext) {

	for _, inout := range r.inouts {
		r.args["_out_"] = utils.Join(inout.out)
		if inout.rspfile != "" {
			r.args["rspfile"] = inout.rspfile
		}

		mainRuleOuts := inout.out
		mainRuleImplicitOuts := inout.implicitOuts
		deps := blueprint.DepsNone

		// ninja currently does not support the case when depfile is
		// defined and multiple outputs at the same time. So adjust the
		// main rule to have a single output, and link the remaining
		// outputs using a separate rule.
		if inout.depfile != "" && (len(inout.out)+len(inout.implicitOuts)) > 1 {
			// No-op rule linking the extra outputs to the main
			// output. Update the extra outputs' mtime in case the
			// script actually creates the extra outputs first.

			allOutputs := append(mainRuleOuts, mainRuleImplicitOuts...)
			mainRuleOuts = allOutputs[0:1]
			mainRuleImplicitOuts = []string{}

			ctx.Build(pctx,
				blueprint.BuildParams{
					Rule:     touchRule,
					Inputs:   mainRuleOuts,
					Outputs:  allOutputs[1:],
					Optional: true,
				})
			deps = blueprint.DepsGCC
		}

		unique_implicits := utils.Unique(append(inout.implicitSrcs, r.implicits...))

		buildparams := blueprint.BuildParams{
			Rule:            *r.rule,
			Inputs:          inout.in,
			Outputs:         mainRuleOuts,
			ImplicitOutputs: mainRuleImplicitOuts,
			Implicits:       unique_implicits,
			Args:            r.args,
			Optional:        true,
			Depfile:         inout.depfile,
			Deps:            deps,
		}

		ctx.Build(pctx, buildparams)
	}
}

func (g *androidNinjaGenerator) generateCommonActions(m *ModuleGenerateCommon, ctx blueprint.ModuleContext, inouts []inout) {
	outputdir := backend.Get().SourceOutputDir(ctx.Module())
	prefixInoutsWithOutputDir(inouts, outputdir)

	cmd, args, implicits, hostTarget := m.getArgs(ctx)
	ldLibraryPath := ""
	if _, ok := args["host_bin"]; ok {
		ldLibraryPath += "LD_LIBRARY_PATH=" + backend.Get().SharedLibsDir(hostTarget) + ":$$LD_LIBRARY_PATH "
	}
	utils.StripUnusedArgs(args, cmd)

	var pool blueprint.Pool
	if proptools.Bool(m.Properties.Console) {
		// Console can be used to run longrunning jobs (even interactive jobs).
		pool = blueprint.Console
	}

	ruleparams := blueprint.RuleParams{
		Command: ldLibraryPath + cmd,
		// Restat is always set to true. This is due to wanting to enable scripts
		// to only update the outputs if they have changed (keeping the same mtime if it
		// has not). If there are no updates, the following rules will not have to update
		// the output.
		Restat:      true,
		Pool:        pool,
		Description: "$out",
	}

	if m.Properties.Rsp_content != nil {
		ruleparams.Rspfile = "${rspfile}"
		ruleparams.RspfileContent = *m.Properties.Rsp_content
	}

	rule := ctx.Rule(pctx, "gen_"+m.Name(), ruleparams,
		append(utils.SortedKeys(args), "depfile", "rspfile", "_out_")...)

	rCtx := ruleContext{
		rule:      &rule,
		inouts:    inouts,
		args:      args,
		implicits: implicits,
	}

	g.buildRules(&rCtx, ctx)
}

func (g *androidNinjaGenerator) install(m interface{}, ctx blueprint.ModuleContext) []string {
	ins := m.(installable)

	props := ins.getInstallableProps()
	installPath, ok := props.getInstallPath()
	if !ok {
		return []string{}
	}
	installPath = filepath.Join("${BuildDir}", installPath)

	installedFiles := []string{}

	rule := installRule
	args := map[string]string{}
	deps := []string{}
	if props.Post_install_cmd != nil {
		rulename := "install"

		cmd := "rm -f $out; cp $in $out ; " + *props.Post_install_cmd

		// Expand args immediately
		cmd = strings.Replace(cmd, "${args}", strings.Join(props.Post_install_args, " "), -1)

		if props.Post_install_tool != nil {
			args["tool"] = *props.Post_install_tool
			deps = append(deps, *props.Post_install_tool)
		}
		utils.StripUnusedArgs(args, cmd)

		rule = ctx.Rule(pctx,
			rulename,
			blueprint.RuleParams{
				Command:     cmd,
				Description: "$out",
			},
			utils.SortedKeys(args)...)
	}

	if provider, ok := m.(file.Provider); ok {
		provider.OutFiles().ForEachIf(
			func(fp file.Path) bool { return fp.IsType(file.TypeInstallable) },
			func(fp file.Path) bool {
				if fp.IsSymLink() {
					symlink := filepath.Join(installPath, fp.UnScopedPath())
					symlinkTgt := filepath.Join(installPath, fp.ExpandLink().UnScopedPath())
					ctx.Build(pctx,
						blueprint.BuildParams{
							Rule:     symlinkRule,
							Outputs:  []string{symlink},
							Inputs:   []string{symlinkTgt},
							Args:     map[string]string{"target": fp.ExpandLink().UnScopedPath()},
							Optional: true,
						})

					installedFiles = append(installedFiles, symlink)
				} else {
					src := fp.BuildPath()
					dest := filepath.Join(installPath, filepath.Base(src))
					// Interpose strip target
					if capable, ok := m.(BackendConfigurationProvider); ok {
						if lib := capable.GetBackendConfiguration(ctx); lib != nil {

							debugPath := lib.getDebugPath()
							separateDebugInfo := debugPath != nil

							debugPathPrefix := installPath //Default to install path
							if separateDebugInfo && *debugPath != "" {
								debugPathPrefix = filepath.Join("${BuildDir}", *debugPath)
							}

							if lib.strip() || separateDebugInfo {
								tc := backend.Get().GetToolchain(lib.getTarget())
								basename := filepath.Base(src)
								strippedSrc := filepath.Join(lib.stripOutputDir(g), basename)
								stArgs := tc.GetStripFlags()
								if lib.strip() {
									stArgs = append(stArgs, "--strip")
								}
								if separateDebugInfo {
									// TODO: This should really be using file interface when enabled
									dbgFile := filepath.Join(debugPathPrefix, basename+".dbg")
									stArgs = append(stArgs, "--debug-file")
									stArgs = append(stArgs, dbgFile)
								}
								stripArgs := map[string]string{
									"args": strings.Join(stArgs, " "),
								}
								ctx.Build(pctx,
									blueprint.BuildParams{
										Rule:     stripRule,
										Outputs:  []string{strippedSrc},
										Inputs:   []string{src},
										Args:     stripArgs,
										Optional: true,
									})
								src = strippedSrc
							}
						}
					}

					ctx.Build(pctx,
						blueprint.BuildParams{
							Rule:      rule,
							Outputs:   []string{dest},
							Inputs:    []string{src},
							Args:      args,
							Implicits: deps,
							Optional:  true,
						})

					installedFiles = append(installedFiles, dest)
				}
				return true
			})
	}

	return append(installedFiles, ins.getInstallDepPhonyNames(ctx)...)
}

func (g *androidNinjaGenerator) generateStrictCommonActions(m *ModuleStrictGenerateCommon, ctx blueprint.ModuleContext, inouts []inout) {
	outputdir := backend.Get().SourceOutputDir(ctx.Module())
	prefixInoutsWithOutputDir(inouts, outputdir)

	cmd, args, implicits, hostLdLibraryPath := m.getArgs(ctx)

	var pool blueprint.Pool
	ruleparams := blueprint.RuleParams{
		Command: hostLdLibraryPath + cmd,
		// Restat is always set to true. This is due to wanting to enable scripts
		// to only update the outputs if they have changed (keeping the same mtime if it
		// has not). If there are no updates, the following rules will not have to update
		// the output.
		Restat:      true,
		Pool:        pool,
		Description: "$out",
	}

	rule := ctx.Rule(pctx, "gen_"+m.Name(), ruleparams,
		append(utils.SortedKeys(args), "depfile", "_out_")...)

	rCtx := ruleContext{
		rule:      &rule,
		inouts:    inouts,
		args:      args,
		implicits: implicits,
	}

	g.buildRules(&rCtx, ctx)
}

func (g *androidNinjaGenerator) generateSourceActions(m *ModuleGenerateSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)

	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

// genruleActions implements generatorBackend.
func (g *androidNinjaGenerator) genruleActions(m *ModuleGenrule, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx)
	g.generateStrictCommonActions(&m.ModuleStrictGenerateCommon, ctx, inouts)

	addPhony(m, ctx, file.GetOutputs(m), !isBuiltByDefault(m))
}

// gensrcsActions implements generatorBackend.
func (*androidNinjaGenerator) gensrcsActions(m *ModuleGensrcs, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// kernelModuleActions implements generatorBackend.
func (*androidNinjaGenerator) kernelModuleActions(m *ModuleKernelObject, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// resourceActions implements generatorBackend.
func (*androidNinjaGenerator) resourceActions(m *ModuleResource, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// sharedActions implements generatorBackend.
func (*androidNinjaGenerator) sharedActions(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// staticActions implements generatorBackend.
func (*androidNinjaGenerator) staticActions(m *ModuleStaticLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// strictBinaryActions implements generatorBackend.
func (*androidNinjaGenerator) strictBinaryActions(m *ModuleStrictBinary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// strictLibraryActions implements generatorBackend.
func (*androidNinjaGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// transformSourceActions implements generatorBackend.
func (*androidNinjaGenerator) transformSourceActions(m *ModuleTransformSource, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// Compile time check for interface `androidNinjaGenerator` being compliant with generatorBackend
var _ generatorBackend = (*androidNinjaGenerator)(nil)
