package core

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
)

var (
	pctx = blueprint.NewPackageContext("bob")

	_ = pctx.VariableFunc("SrcDir", func(interface{}) (string, error) {
		return getSourceDir(), nil
	})
	_ = pctx.VariableFunc("BuildDir", func(interface{}) (string, error) {
		return getBuildDir(), nil
	})
	_ = pctx.VariableFunc("BobScriptsDir", func(interface{}) (string, error) {
		return getBobScriptsDir(), nil
	})

	enableToc = getTocUsageFromEnvironment()
)

type linuxGenerator struct {
}

/* Compile time checks for interfaces that must be implemented by linuxGenerator */
var _ generatorBackend = (*linuxGenerator)(nil)

func getTocUsageFromEnvironment() bool {
	enable := true // Default to using toc files
	if str, ok := os.LookupEnv("BOB_ALWAYS_LINK_SHARED_LIBS"); ok {
		// Disable according to the environment variable
		//
		// Be permissive in the values accepted to disable this
		// feature. If someone is trying to set this variable, then by
		// definition they are looking to disable it. Users who want
		// the default behavior are unlikely to set it. So look for a
		// few values which might be used to indicate "I'd like the
		// default behavior", and take any other value to mean change
		// behavior.
		//
		// This should reduce queries about what's the right setting
		// to use to disable toc usage.
		if !utils.Contains([]string{"n", "N", "0", ""}, str) {
			enable = false
		}
	}
	return enable
}

func addPhony(p phonyInterface, ctx blueprint.ModuleContext,
	installDeps []string, optional bool) {

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     blueprint.Phony,
			Inputs:   installDeps,
			Outputs:  []string{p.shortName()},
			Optional: optional,
		})
}

func (g *linuxGenerator) getPhonyFiles(p dependentInterface) []string {
	return utils.NewStringSlice(p.outputs(), p.implicitOutputs())
}

type singleOutputModule interface {
	blueprint.Module
	outputName() string
	outputFileName() string
}

type targetableModule interface {
	singleOutputModule
	getTarget() toolchain.TgtType
}

// Modules implementing sharedLibProducer create a shared library
type sharedLibProducer interface {
	targetableModule
	getTocName() string
}

// Modules implementing the linkableModule interface are linked
// by `ld` to produce a shared library or binary.
type linkableModule interface {
	getVersionScript(ctx blueprint.ModuleContext) *string
	// GetWholeStaticLibs(ctx blueprint.ModuleContext) []string
	GetStaticLibs(ctx blueprint.ModuleContext) []string
}

// Full path for shared libraries, in a shared location to simplify linking.
// As long as the module is targetable, we can infer the library path.
func (g *linuxGenerator) getSharedLibLinkPath(t targetableModule) string {
	// TODO: this should be part of core/backend
	return filepath.Join(backend.Get().SharedLibsDir(t.getTarget()), t.outputFileName())
}

var _ = pctx.StaticVariable("toc", "${BobScriptsDir}/library_toc.py")
var tocRule = pctx.StaticRule("shared_library_toc",
	blueprint.RuleParams{
		Command:     "$toc $in -o $out $tocflags",
		CommandDeps: []string{"$toc"},
		Description: "Generate toc $out",
		Restat:      true,
	},
	"tocflags")

func (g *linuxGenerator) addSharedLibToc(ctx blueprint.ModuleContext, soFile, tocFile string, tgt toolchain.TgtType) {
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

func (*linuxGenerator) aliasActions(a *ModuleAlias, ctx blueprint.ModuleContext) {
	srcs := []string{}

	/* Only depend on enabled targets */
	ctx.VisitDirectDepsIf(
		func(p blueprint.Module) bool { return ctx.OtherModuleDependencyTag(p) == AliasTag },
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

var _ = pctx.StaticVariable("strip", "${BobScriptsDir}/strip.py")
var stripRule = pctx.StaticRule("strip",
	blueprint.RuleParams{
		Command:     "$strip $args -o $out $in",
		CommandDeps: []string{"$strip"},
		Description: "strip $out",
	}, "args")

var installRule = pctx.StaticRule("install",
	blueprint.RuleParams{
		Command:     "rm -f $out; cp $in $out",
		Description: "$out",
	})

func (g *linuxGenerator) install(m interface{}, ctx blueprint.ModuleContext) []string {
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

	for _, src := range ins.filesToInstall(ctx) {
		dest := filepath.Join(installPath, filepath.Base(src))

		// Interpose strip target
		if lib, ok := m.(stripable); ok {
			debugPath := lib.getDebugPath()
			separateDebugInfo := debugPath != nil
			if separateDebugInfo {
				if *debugPath == "" {
					// Install next to library by default
					debugPath = &installPath
				} else {
					*debugPath = filepath.Join("${BuildDir}", *debugPath)
				}
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
					dbgFile := filepath.Join(*debugPath, basename+".dbg")
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

	if provider, ok := m.(FileProvider); ok {
		provider.OutFiles().ForEachIf(
			func(fp file.Path) bool { return fp.IsSymLink() },
			func(fp file.Path) bool {
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
				return true
			})
	}

	return append(installedFiles, ins.getInstallDepPhonyNames(ctx)...)
}

func (g *linuxGenerator) resourceActions(r *ModuleResource, ctx blueprint.ModuleContext) {
	installDeps := g.install(r, ctx)
	addPhony(r, ctx, installDeps, false)
}

func (g *linuxGenerator) filegroupActions(m *ModuleFilegroup, ctx blueprint.ModuleContext) {

}
