/*
 * Copyright 2018-2020 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

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
	toolchainSet
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

	deps := utils.NewStringSlice(p.outputs(), p.implicitOutputs(), installDeps)

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     blueprint.Phony,
			Inputs:   deps,
			Outputs:  []string{p.shortName()},
			Optional: optional,
		})
}

func (g *linuxGenerator) escapeFlag(s string) string {
	return proptools.NinjaAndShellEscape(s)
}

func (g *linuxGenerator) sourceDir() string {
	return "${SrcDir}"
}

func (g *linuxGenerator) buildDir() string {
	return "${BuildDir}"
}

func (g *linuxGenerator) bobScriptsDir() string {
	return "${BobScriptsDir}"
}

func (g *linuxGenerator) sourceOutputDir(m *generateCommon) string {
	return filepath.Join("${BuildDir}", "gen", m.Name())
}

type singleOutputModule interface {
	blueprint.Module
	outputName() string
	outputFileName() string
}

type targetableModule interface {
	singleOutputModule
	getTarget() tgtType
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
	GetWholeStaticLibs(ctx blueprint.ModuleContext) []string
	GetStaticLibs(ctx blueprint.ModuleContext) []string
}

func (g *linuxGenerator) staticLibOutputDir(m *staticLibrary) string {
	return filepath.Join("${BuildDir}", string(m.Properties.TargetType), "static")
}

func (g *linuxGenerator) sharedLibsDir(tgt tgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "shared")
}

// Full path for shared libraries, in a shared location to simplify linking.
// As long as the module is targetable, we can infer the library path.
func (g *linuxGenerator) getSharedLibLinkPath(t targetableModule) string {
	return filepath.Join(g.sharedLibsDir(t.getTarget()), t.outputFileName())
}

// Full path for shared library tables of content.
// As long as the module is targetable, we can infer the library path.
func (g *linuxGenerator) getSharedLibTocPath(l sharedLibProducer) string {
	return filepath.Join(g.sharedLibsDir(l.getTarget()), l.getTocName())
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

func (g *linuxGenerator) addSharedLibToc(ctx blueprint.ModuleContext, soFile, tocFile string, tgt tgtType) {
	tc := g.getToolchain(tgt)
	tocFlags := tc.getLibraryTocFlags()

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     tocRule,
			Outputs:  []string{tocFile},
			Inputs:   []string{soFile},
			Optional: true,
			Args:     map[string]string{"tocflags": strings.Join(tocFlags, " ")},
		})
}

func (g *linuxGenerator) binaryOutputDir(tgt tgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "executable")
}

// Full path for a generated binary. This ensures generated binaries
// are available in the same directory as compiled binaries
func (g *linuxGenerator) getBinaryPath(t targetableModule) string {
	return filepath.Join(g.binaryOutputDir(t.getTarget()), t.outputFileName())
}

func (*linuxGenerator) aliasActions(m *alias, ctx blueprint.ModuleContext) {
	srcs := []string{}

	/* Only depend on enabled targets */
	ctx.VisitDirectDepsIf(
		func(p blueprint.Module) bool { return ctx.OtherModuleDependencyTag(p) == aliasTag },
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
			Outputs:  []string{m.Name()},
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

		args["bob_config"] = configFile
		args["bob_config_json"] = filepath.Join(getBuildDir(), configJSONFile)
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

	// Check if this is a resource
	_, isResource := ins.(*resource)

	for _, src := range ins.filesToInstall(ctx) {
		dest := filepath.Join(installPath, filepath.Base(src))
		// Resources always come from the source directory.
		// All other module types install files from the build directory.
		if isResource {
			src = getBackendPathInSourceDir(g, src)
		}

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
				tc := g.getToolchain(lib.getTarget())
				basename := filepath.Base(src)
				strippedSrc := filepath.Join(lib.stripOutputDir(g), basename)
				stArgs := tc.getStripFlags()
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

	if symlinkIns, ok := m.(symlinkInstaller); ok {
		symlinks := symlinkIns.librarySymlinks(ctx)

		for key, value := range symlinks {
			symlink := filepath.Join(installPath, key)
			symlinkTgt := filepath.Join(installPath, value)
			ctx.Build(pctx,
				blueprint.BuildParams{
					Rule:     symlinkRule,
					Outputs:  []string{symlink},
					Inputs:   []string{symlinkTgt},
					Args:     map[string]string{"target": value},
					Optional: true,
				})

			installedFiles = append(installedFiles, symlink)
		}
	}

	return append(installedFiles, ins.getInstallDepPhonyNames(ctx)...)
}

func (g *linuxGenerator) resourceActions(m *resource, ctx blueprint.ModuleContext) {
	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, false)
}

func (g *linuxGenerator) init(ctx *blueprint.Context, config *bobConfig) {
	g.toolchainSet.parseConfig(config)
}
