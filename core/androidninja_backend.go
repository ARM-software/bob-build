package core

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type androidNinjaGenerator struct {
}

// aliasActions implements generatorBackend.
func (*androidNinjaGenerator) aliasActions(a *ModuleAlias, ctx blueprint.ModuleContext) {
	srcs := []string{}

	/* Only depend on enabled targets */
	ctx.VisitDirectDepsIf(
		func(p blueprint.Module) bool { return ctx.OtherModuleDependencyTag(p) == tag.AliasTag },
		func(p blueprint.Module) {
			if e, ok := p.(enableable); ok {
				if !isEnabled(e) {
					return
				}
			}
			name := ctx.OtherModuleName(p)
			if lib, ok := p.(phonyInterface); ok {
				name = lib.shortName()
			}

			srcs = append(srcs, name)
		})

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     blueprint.Phony,
			Inputs:   srcs,
			Outputs:  []string{a.Name()},
			Optional: true,
		})
}

// binaryActions implements generatorBackend.
func (g *androidNinjaGenerator) binaryActions(m *ModuleBinary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objectFiles, nonCompiledDeps := g.CompileObjs(m, ctx, tc)
	/* By default, build all target binaries */
	optional := !isBuiltByDefault(m)

	buildWrapperDeps := []string{}
	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		_, buildWrapperDeps = bc.GetBuildWrapperAndDeps(ctx)
	}

	orderOnly := buildWrapperDeps
	if enableToc {
		// Add an order only dependecy on the actual libraries to cover
		// the case where the .so is deleted but the toc is still
		// present.
		orderOnly = append(orderOnly, g.getSharedLibLinkPaths(ctx)...)
	}

	// TODO: Propogate shared library orderOnly dependencies correctly
	// if m.Name() == "shared_strict_lib_binary" {
	// 	orderOnly = []string{"lib_simple.so"}
	// }

	outs := m.OutFiles().ToStringSliceIf(
		func(p file.Path) bool { return p.IsType(file.TypeBinary) },
		func(p file.Path) string { return p.BuildPath() })

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      executableRule,
			Outputs:   outs,
			Inputs:    objectFiles,
			Implicits: append(g.ccLinkImplicits(m, ctx, enableToc), nonCompiledDeps...),
			OrderOnly: orderOnly,
			Optional:  true,
			Args:      g.getCommonLibArgs(m, ctx),
		})

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, optional)
}

func (g *androidNinjaGenerator) getCommonLibArgs(m BackendCommonLibraryInterface, ctx blueprint.ModuleContext) map[string]string {
	tc := backend.Get().GetToolchain(m.getTarget())

	ldflags := m.FlagsIn().Filtered(func(f flag.Flag) bool {
		return f.MatchesType(flag.TypeLinker)
	}).ToStringSlice()

	ldlibs := m.FlagsIn().Filtered(func(f flag.Flag) bool {
		return f.MatchesType(flag.TypeLinkLibrary)
	}).ToStringSlice()

	m.FlagsInTransitive(ctx).ForEachIf(
		func(f flag.Flag) bool {
			return f.MatchesType(flag.TypeTransitiveLinker)
		},
		func(f flag.Flag) {
			ldlibs = append(ldlibs, f.ToString())
		},
	)

	if m.IsForwardingSharedLibrary() {
		ldflags = append(ldflags, tc.GetLinker().KeepUnusedDependencies())
	} else {
		ldflags = append(ldflags, tc.GetLinker().DropUnusedDependencies())
	}

	versionScript := m.getVersionScript(ctx)
	if versionScript != nil {
		ldflags = append(ldflags, tc.GetLinker().SetVersionScript(*versionScript))
	}

	sharedLibLdlibs, sharedLibLdflags := g.getSharedLibFlags(m, ctx)

	linker := tc.GetLinker().GetTool()
	tcLdflags := tc.GetLinker().GetFlags()
	tcLdlibs := tc.GetLinker().GetLibs()

	wholeStaticLibs := GetWholeStaticLibs(ctx)
	staticLibs := m.GetStaticLibs(ctx)
	staticLibFlags := []string{}
	if len(wholeStaticLibs) > 0 {
		staticLibFlags = append(staticLibFlags, tc.GetLinker().LinkWholeArchives(
			wholeStaticLibs))
	}
	staticLibFlags = append(staticLibFlags, staticLibs...)
	sharedLibDir := backend.Get().SharedLibsDir(m.getTarget())

	args := map[string]string{
		"build_wrapper":   "",
		"ldflags":         utils.Join(tcLdflags, ldflags, sharedLibLdflags),
		"linker":          linker,
		"shared_libs_dir": sharedLibDir,
		"shared_libs_flags": utils.Join(append(sharedLibLdlibs,
			tc.GetLinker().SetRpathLink(sharedLibDir))),
		"static_libs": utils.Join(staticLibFlags),
		"ldlibs":      utils.Join(ldlibs, tcLdlibs),
	}

	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		args["build_wrapper"], _ = bc.GetBuildWrapperAndDeps(ctx)
	}

	return args
}

func (g *androidNinjaGenerator) getSharedLibFlags(m BackendCommonLibraryInterface, ctx blueprint.ModuleContext) (ldlibs []string, ldflags []string) {
	// With forwarding shared library we do not have to use
	// --no-as-needed for dependencies because it is already set
	useNoAsNeeded := !m.IsForwardingSharedLibrary()
	hasForwardingLib := false
	libPaths := []string{}
	tc := backend.Get().GetToolchain(m.getTarget())

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.SharedTag },
		func(m blueprint.Module) {
			if sl, ok := m.(*ModuleSharedLibrary); ok {
				b := &sl.ModuleLibrary.Properties.Build
				if b.isForwardingSharedLibrary() {
					hasForwardingLib = true
					ldlibs = append(ldlibs, tc.GetLinker().KeepSharedLibraryTransitivity())
					if useNoAsNeeded {
						ldlibs = append(ldlibs, tc.GetLinker().KeepUnusedDependencies())
					}
				}
				ldlibs = append(ldlibs, pathToLibFlag(sl.outputName()))
				if b.isForwardingSharedLibrary() {
					if useNoAsNeeded {
						ldlibs = append(ldlibs, tc.GetLinker().DropUnusedDependencies())
					}
					ldlibs = append(ldlibs, tc.GetLinker().DropSharedLibraryTransitivity())
				}
				if installPath, ok := sl.Properties.InstallableProps.getInstallPath(); ok {
					libPaths = utils.AppendIfUnique(libPaths, installPath)
				}
			} else if sl, ok := m.(*generateSharedLibrary); ok {
				ldlibs = append(ldlibs, pathToLibFlag(sl.outputName()))
				if installPath, ok := sl.ModuleGenerateCommon.Properties.InstallableProps.getInstallPath(); ok {
					libPaths = utils.AppendIfUnique(libPaths, installPath)
				}
			} else if el, ok := m.(*ModuleExternalLibrary); ok {
				ldlibs = append(ldlibs, el.FlagsOut().Filtered(func(f flag.Flag) bool {
					return f.MatchesType(flag.TypeLinkLibrary)
				}).ToStringSlice()...)

				ldflags = append(ldflags, el.FlagsOut().Filtered(func(f flag.Flag) bool {
					return f.MatchesType(flag.TypeLinker)
				}).ToStringSlice()...)
			} else if sl, ok := m.(*ModuleStrictLibrary); ok {
				ldlibs = append(ldlibs, pathToLibFlag(sl.Name()+".so"))
			} else {
				utils.Die("%s is not a shared library", ctx.OtherModuleName(m))
			}
		})

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.ToolchainTag },
		func(m blueprint.Module) {
			if t, ok := m.(*ModuleToolchain); ok {
				ldflags = append(ldflags, t.FlagsOut().Filtered(func(f flag.Flag) bool {
					return f.MatchesType(flag.TypeLinker)
				}).ToStringSlice()...)
			}
		})

	if hasForwardingLib {
		ldlibs = append(ldlibs, tc.GetLinker().GetForwardingLibFlags())
	}
	if m.IsRpathWanted() {
		props := m.getInstallableProps()
		if installPath, ok := props.getInstallPath(); ok {
			var rpaths []string
			for _, path := range libPaths {
				out, err := filepath.Rel(installPath, path)
				if err != nil {
					utils.Die("Could not find relative path for: %s due to: %s", path, err)
				}
				rpaths = append(rpaths, "'$$ORIGIN/"+out+"'")
			}
			ldlibs = append(ldlibs, tc.GetLinker().SetRpath(rpaths))
		}
	}

	// https://stackoverflow.com/questions/47279824/android-ndk-dlopen-failed/48291044#48291044
	if l, ok := m.(BackendCommonSharedLibraryInterface); ok {
		ldlibs = append(ldlibs, "-Wl,-soname,"+l.getRealName())
	}

	return
}

func (g *androidNinjaGenerator) ccLinkImplicits(l linkableModule, ctx blueprint.ModuleContext, useToc bool) []string {
	implicits := utils.NewStringSlice(GetWholeStaticLibs(ctx), l.GetStaticLibs(ctx))
	if useToc {
		implicits = append(implicits, g.getSharedLibTocPaths(ctx)...)
	} else {
		implicits = append(implicits, g.getSharedLibLinkPaths(ctx)...)
	}
	versionScript := l.getVersionScript(ctx)
	if versionScript != nil {
		implicits = append(implicits, *versionScript)
	}

	return implicits
}

func (g *androidNinjaGenerator) getSharedLibTocPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.SharedTag },
		func(m blueprint.Module) {
			if _, ok := m.(sharedLibProducer); ok { //Remove this check and replace it with an API call
				if m, ok := m.(file.Provider); ok {
					if toc, ok := m.OutFiles().FindSingle(
						func(p file.Path) bool { return p.IsType(file.TypeToc) }); ok {
						libs = append(libs, toc.BuildPath())
					}
				}
			} else if _, ok := m.(*ModuleExternalLibrary); ok {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else {
				utils.Die("%s doesn't produce a shared library", ctx.OtherModuleName(m))
			}
		})
	return
}

func (g *androidNinjaGenerator) getSharedLibLinkPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.SharedTag },
		func(m blueprint.Module) {
			if t, ok := m.(targetableModule); ok {
				libs = append(libs, g.getSharedLibLinkPath(t))
			} else if _, ok := m.(*ModuleExternalLibrary); ok {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else {
				utils.Die("%s doesn't support targets", ctx.OtherModuleName(m))
			}
		})
	return
}

func (g *androidNinjaGenerator) CompileObjs(l Compilable, ctx blueprint.ModuleContext, tc toolchain.Toolchain) ([]string, []string) {
	orderOnly := GetGeneratedHeadersFiles(ctx)

	// tc := backend.Get().GetToolchain(tgtType)
	as, astargetflags := tc.GetAssembler()
	cc, cctargetflags := tc.GetCCompiler()
	cxx, cxxtargetflags := tc.GetCXXCompiler()
	cflagsList := []string{}

	// Get all the required flags and group them into includes and everything else.
	// This should make it easier to visually inspect the flags in logs/ninja files.
	l.FlagsInTransitive(ctx).GroupByType(flag.TypeInclude).ForEach(
		func(f flag.Flag) {
			switch {
			case (f.Type() & flag.TypeCompilable) == flag.TypeC: //c exclusive flags
				cctargetflags = append(cctargetflags, f.ToString())
			case f.MatchesType(flag.TypeCC | flag.TypeInclude):
				cflagsList = append(cflagsList, f.ToString())
			case f.MatchesType(flag.TypeAsm):
				astargetflags = append(astargetflags, f.ToString())
			case f.MatchesType(flag.TypeCpp):
				cxxtargetflags = append(cxxtargetflags, f.ToString())
			}
		},
	)

	ctx.Variable(pctx, "asflags", strings.Join(astargetflags, " "))
	ctx.Variable(pctx, "cflags", strings.Join(cflagsList, " "))
	ctx.Variable(pctx, "conlyflags", strings.Join(cctargetflags, " "))
	ctx.Variable(pctx, "cxxflags", strings.Join(cxxtargetflags, " "))

	objectFiles := []string{}
	nonCompiledDeps := []string{}

	// TODO: use tags here instead of extensions
	l.GetFiles(ctx).ForEach(
		func(source file.Path) bool {
			var rule blueprint.Rule
			args := make(map[string]string)
			switch source.Ext() {
			case ".s":
				args["ascompiler"] = as
				args["asflags"] = "$asflags"
				rule = asRule
			case ".S":
				// Assembly with .S suffix must be preprocessed by the C compiler
				fallthrough
			case ".c":
				args["ccompiler"] = cc
				args["cflags"] = "$cflags"
				args["conlyflags"] = "$conlyflags"
				rule = ccRule
			case ".cc":
				fallthrough
			case ".cpp":
				args["cxxcompiler"] = cxx
				args["cflags"] = "$cflags"
				args["cxxflags"] = "$cxxflags"
				rule = cxxRule
			default:
				nonCompiledDeps = append(nonCompiledDeps, source.BuildPath())
				return true
			}

			bc := GetModuleBackendConfiguration(ctx, l)

			buildWrapperDeps := []string{}
			if bc != nil {
				args["build_wrapper"], buildWrapperDeps = bc.GetBuildWrapperAndDeps(ctx)
			} else {
				args["build_wrapper"] = ""
			}

			output := g.ObjDir(l) + source.RelBuildPath() + ".o"

			ctx.Build(pctx,
				blueprint.BuildParams{
					Rule:      rule,
					Outputs:   []string{output},
					Inputs:    []string{source.BuildPath()},
					Args:      args,
					OrderOnly: utils.NewStringSlice(orderOnly, buildWrapperDeps),
					Optional:  true,
				})
			objectFiles = append(objectFiles, output)

			return true
		})

	return objectFiles, nonCompiledDeps
}

func (g *androidNinjaGenerator) ObjDir(m Compilable) string {
	return filepath.Join("${BuildDir}", string(m.getTarget()), "objects", m.outputName()) + string(os.PathSeparator)
}

// executableTestActions implements generatorBackend.
func (*androidNinjaGenerator) executableTestActions(m *ModuleTest, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// filegroupActions implements generatorBackend.
func (*androidNinjaGenerator) filegroupActions(m *ModuleFilegroup, ctx blueprint.ModuleContext) {

}

// genBinaryActions implements generatorBackend.
func (g *androidNinjaGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	// Create a rule to copy the generated binary
	// from gen_dir to the common binary directory
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:   copyRule,
			Inputs: file.GetOutputs(m),
			Outputs: []string{filepath.Join(
				backend.Get().BinaryOutputDir(m.getTarget()),
				m.outputFileName())},
			Optional: true,
		})

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

// genSharedActions implements generatorBackend.
func (g *androidNinjaGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	// Create a rule to copy the generated library
	// from gen_dir to the common library directory
	soFile := g.getSharedLibLinkPath(m)
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     copyRule,
			Inputs:   file.GetOutputs(m),
			Outputs:  []string{soFile},
			Optional: true,
		})

	if toc, ok := m.OutFiles().FindSingle(
		func(p file.Path) bool { return p.IsType(file.TypeToc) }); ok {
		g.addSharedLibToc(ctx, soFile, toc.BuildPath(), m.getTarget())
	}

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *androidNinjaGenerator) getSharedLibLinkPath(t targetableModule) string {
	return filepath.Join(backend.Get().SharedLibsDir(t.getTarget()), t.outputFileName())
}

func (g *androidNinjaGenerator) addSharedLibToc(ctx blueprint.ModuleContext, soFile, tocFile string, tgt toolchain.TgtType) {
	tc := backend.Get().GetToolchain(tgt)
	tocFlags := tc.GetLibraryTocFlags()

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     tocRule,
			Outputs:  []string{tocFile},
			Inputs:   []string{soFile},
			Optional: true,
			Args:     map[string]string{"tocflags": strings.Join(tocFlags, " ")},
		})
}

// genStaticActions implements generatorBackend.
func (g *androidNinjaGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
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
func (g *androidNinjaGenerator) gensrcsActions(gr *ModuleGensrcs, ctx blueprint.ModuleContext) {
	inouts := gr.generateInouts(ctx)
	g.generateStrictCommonActions(&gr.ModuleStrictGenerateCommon, ctx, inouts)

	addPhony(gr, ctx, file.GetOutputs(gr), !isBuiltByDefault(gr))
}

// kernelModuleActions implements generatorBackend.
func (*androidNinjaGenerator) kernelModuleActions(m *ModuleKernelObject, ctx blueprint.ModuleContext) {
	GetLogger().Warn(warnings.AndroidOutOfTreeUnsupportedModule, ctx.BlueprintsFile(), ctx.ModuleName())
}

// resourceActions implements generatorBackend.
func (g *androidNinjaGenerator) resourceActions(m *ModuleResource, ctx blueprint.ModuleContext) {
	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, false)
}

// sharedActions implements generatorBackend.
func (g *androidNinjaGenerator) sharedActions(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.getTarget())
	objs, implicits := g.CompileObjs(m, ctx, tc)

	installDeps := g.install(m, ctx)
	g.SharedSymlinkActions(ctx, m)
	g.SharedLinkActions(ctx, m, tc, objs, implicits)
	g.SharedTocActions(ctx, m)

	installDeps = append(installDeps, file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *androidNinjaGenerator) SharedLinkActions(ctx blueprint.ModuleContext,
	m BackendCommonLibraryInterface,
	tc toolchain.Toolchain,
	objs []string, implicits []string) {

	buildWrapperDeps := []string{}
	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		_, buildWrapperDeps = bc.GetBuildWrapperAndDeps(ctx)
	}

	orderOnly := buildWrapperDeps
	if enableToc {
		// Add an order only dependecy on the actual libraries to cover
		// the case where the .so is deleted but the toc is still
		// present.
		orderOnly = append(orderOnly, g.getSharedLibLinkPaths(ctx)...)
	}

	outs := m.OutFiles().ToStringSliceIf(
		func(p file.Path) bool { return p.IsType(file.TypeShared) && !p.IsSymLink() },
		func(p file.Path) string { return p.BuildPath() })

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      sharedLibraryRule,
			Outputs:   outs,
			Inputs:    objs,
			Implicits: append(g.ccLinkImplicits(m, ctx, enableToc), implicits...),
			OrderOnly: orderOnly,
			Optional:  true,
			Args:      g.getCommonLibArgs(m, ctx),
		})

}

func (g *androidNinjaGenerator) SharedTocActions(ctx blueprint.ModuleContext,
	m BackendCommonSharedLibraryInterface) {
	if toc, ok := m.OutFiles().FindSingle(
		func(p file.Path) bool { return p.IsType(file.TypeToc) }); ok {
		outputdir := backend.Get().SharedLibsDir(m.getTarget())
		soFile := filepath.Join(outputdir, m.getRealName())
		g.addSharedLibToc(ctx, soFile, toc.BuildPath(), m.getTarget())
	}
}

func (g *androidNinjaGenerator) SharedSymlinkActions(ctx blueprint.ModuleContext,
	m BackendCommonLibraryInterface) (deps []string) {

	m.OutFiles().ForEachIf(
		func(fp file.Path) bool { return fp.IsSymLink() },
		func(fp file.Path) bool {
			ctx.Build(pctx,
				blueprint.BuildParams{
					Rule:     symlinkRule,
					Inputs:   []string{fp.ExpandLink().BuildPath()},
					Outputs:  []string{fp.BuildPath()},
					Args:     map[string]string{"target": fp.ExpandLink().UnScopedPath()},
					Optional: true,
				})
			deps = append(deps, fp.BuildPath())
			return true
		})

	return
}

// staticActions implements generatorBackend.
func (g *androidNinjaGenerator) staticActions(m *ModuleStaticLibrary, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	// The archiver rules do not allow adding arguments that the user can
	// set, so does not support nonCompiledDeps
	objectFiles, _ := g.CompileObjs(m, ctx, tc)

	g.ArchivableActions(ctx, m, tc, objectFiles)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *androidNinjaGenerator) ArchivableActions(ctx blueprint.ModuleContext,
	m Archivable,
	tc toolchain.Toolchain,
	objs []string) {
	wholeStaticLibs := GetWholeStaticLibs(ctx)

	rule := staticLibraryRule
	arBinary, _ := tc.GetArchiver()

	args := map[string]string{
		"ar":            arBinary,
		"build_wrapper": "",
	}

	bc := GetModuleBackendConfiguration(ctx, m)
	buildWrapperDeps := []string{}
	if bc != nil {
		args["build_wrapper"], buildWrapperDeps = bc.GetBuildWrapperAndDeps(ctx)

	}

	implicits := wholeStaticLibs

	if len(wholeStaticLibs) > 0 {
		rule = wholeStaticLibraryRule
		args["whole_static_libs"] = strings.Join(wholeStaticLibs, " ")
	} else if len(objs) == 0 && getConfig(ctx).Properties.GetBool("osx") {
		// OSX workaround, see rule for details.
		rule = emptyStaticLibraryRule
		// To create an empty lib, we require a dummy object file,
		// we use the detected compiler to emit it.
		cc, _ := tc.GetCCompiler()
		args["ccompiler"] = cc
	}

	outs := m.OutFiles().ToStringSliceIf(
		func(p file.Path) bool { return p.IsType(file.TypeArchive) },
		func(p file.Path) string { return p.BuildPath() })

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      rule,
			Outputs:   outs,
			Inputs:    objs,
			Implicits: implicits,
			OrderOnly: buildWrapperDeps,
			Optional:  true,
			Args:      args,
		})
}

// strictBinaryActions implements generatorBackend.
func (g *androidNinjaGenerator) strictBinaryActions(m *ModuleStrictBinary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objectFiles, nonCompiledDeps := g.CompileObjs(m, ctx, tc)
	/* By default, build all target binaries */
	optional := !isBuiltByDefault(m)

	buildWrapperDeps := []string{}
	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		_, buildWrapperDeps = bc.GetBuildWrapperAndDeps(ctx)
	}

	orderOnly := buildWrapperDeps
	if enableToc {
		// Add an order only dependecy on the actual libraries to cover
		// the case where the .so is deleted but the toc is still
		// present.
		orderOnly = append(orderOnly, g.getSharedLibLinkPaths(ctx)...)
	}

	// TODO: Propogate shared library orderOnly dependencies correctly
	// if m.Name() == "shared_strict_lib_binary" {
	// 	orderOnly = []string{"lib_simple.so"}
	// }

	outs := m.OutFiles().ToStringSliceIf(
		func(p file.Path) bool { return p.IsType(file.TypeBinary) },
		func(p file.Path) string { return p.BuildPath() })

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      executableRule,
			Outputs:   outs,
			Inputs:    objectFiles,
			Implicits: append(g.ccLinkImplicits(m, ctx, enableToc), nonCompiledDeps...),
			OrderOnly: orderOnly,
			Optional:  true,
			Args:      g.getCommonLibArgs(m, ctx),
		})

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, optional)
}

// strictLibraryActions implements generatorBackend.
func (g *androidNinjaGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objs, implicits := g.CompileObjs(m, ctx, tc)

	g.SharedLinkActions(ctx, m, tc, objs, implicits)
	g.SharedTocActions(ctx, m)

	g.ArchivableActions(ctx, m, tc, objs)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	installDeps = append(installDeps, g.SharedSymlinkActions(ctx, m)...)

	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

// transformSourceActions implements generatorBackend.
func (g *androidNinjaGenerator) transformSourceActions(m *ModuleTransformSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

// Compile time check for interface `androidNinjaGenerator` being compliant with generatorBackend
var _ generatorBackend = (*androidNinjaGenerator)(nil)
