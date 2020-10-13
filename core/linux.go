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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
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
)

type linuxGenerator struct {
	toolchainSet
}

/* Compile time checks for interfaces that must be implemented by linuxGenerator */
var _ generatorBackend = (*linuxGenerator)(nil)

// Convert a path to a library into a compiler flag.
// This needs to strip any path, file extension, lib prefix, and prepend -l
func pathToLibFlag(path string) string {
	_, base := filepath.Split(path)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	if !strings.HasPrefix(base, "lib") {
		panic(errors.New("Shared library name must start with 'lib' prefix"))
	}
	base = strings.TrimPrefix(base, "lib")
	return "-l" + base
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

// Where to put generated shared libraries to simplify linking
// As long as the module is targetable, we can infer the library path
func getSharedLibLinkPath(t targetableModule) string {
	return filepath.Join("${BuildDir}", string(t.getTarget()), "shared", t.outputFileName())
}

// Where to put generated binaries in order to make sure generated binaries
// are available in the same directory as compiled binaries
func getBinaryPath(t targetableModule) string {
	return filepath.Join("${BuildDir}", string(t.getTarget()), "executable", t.outputFileName())
}

var asRule = pctx.StaticRule("as",
	blueprint.RuleParams{
		Depfile:     "$out.d",
		Deps:        blueprint.DepsGCC,
		Command:     "$build_wrapper $ascompiler $asflags $in -MD $depfile -o $out",
		Description: "$out",
	}, "ascompiler", "asflags", "build_wrapper", "depfile")

var ccRule = pctx.StaticRule("cc",
	blueprint.RuleParams{
		Depfile:     "$out.d",
		Deps:        blueprint.DepsGCC,
		Command:     "$build_wrapper $ccompiler -c $cflags $conlyflags -MMD -MF $depfile $in -o $out",
		Description: "$out",
	}, "ccompiler", "cflags", "conlyflags", "build_wrapper", "depfile")

var cxxRule = pctx.StaticRule("cxx",
	blueprint.RuleParams{
		Depfile:     "$out.d",
		Deps:        blueprint.DepsGCC,
		Command:     "$build_wrapper $cxxcompiler -c $cflags $cxxflags -MMD -MF $depfile $in -o $out",
		Description: "$out",
	}, "cxxcompiler", "cflags", "cxxflags", "build_wrapper", "depfile")

func (l *library) ObjDir() string {
	return filepath.Join("${BuildDir}", string(l.Properties.TargetType), "objects", l.outputName()) + string(os.PathSeparator)
}

// This function has common support to compile objs for static libs, shared libs and binaries.
func (l *library) CompileObjs(ctx blueprint.ModuleContext) ([]string, []string) {
	g := getBackend(ctx)
	srcs := l.GetSrcs(ctx)

	expLocalIncludes, expIncludes, exportedCflags := l.GetExportedVariables(ctx)
	// There are 2 sets of include dirs - "global" and "local".
	// Local acts on the root source directory.

	// The order we want is  local_include_dirs, export_local_include_dirs,
	//                       include_dirs, export_include_dirs
	localIncludeDirs := utils.NewStringSlice(l.Properties.Local_include_dirs,
		l.Properties.Export_local_include_dirs)

	// Prefix all local includes with SrcDir
	localIncludeDirs = utils.PrefixDirs(localIncludeDirs, "${SrcDir}")
	expLocalIncludes = utils.PrefixDirs(expLocalIncludes, "${SrcDir}")

	includeDirs := append(localIncludeDirs, l.Properties.Include_dirs...)
	includeDirs = append(includeDirs, l.Properties.Export_include_dirs...)
	includeDirs = append(includeDirs, expLocalIncludes...)
	includeDirs = append(includeDirs, expIncludes...)

	gendirs, orderOnly := l.GetGeneratedHeaders(ctx)
	includeDirs = append(includeDirs, gendirs...)
	includeFlags := utils.PrefixAll(includeDirs, "-I")
	cflagsList := utils.NewStringSlice(l.Properties.Cflags, l.Properties.Export_cflags,
		exportedCflags, includeFlags)

	tc := g.getToolchain(l.Properties.TargetType)
	as, astargetflags := tc.getAssembler()
	cc, cctargetflags := tc.getCCompiler()
	cxx, cxxtargetflags := tc.getCXXCompiler()

	ctx.Variable(pctx, "asflags", utils.Join(astargetflags, l.Properties.Asflags))
	ctx.Variable(pctx, "cflags", utils.Join(cflagsList))
	ctx.Variable(pctx, "conlyflags", utils.Join(cctargetflags, l.Properties.Conlyflags))
	ctx.Variable(pctx, "cxxflags", utils.Join(cxxtargetflags, l.Properties.Cxxflags))

	objectFiles := []string{}
	nonCompiledDeps := []string{}

	for _, source := range srcs {
		var rule blueprint.Rule
		args := make(map[string]string)
		switch path.Ext(source) {
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
			nonCompiledDeps = append(nonCompiledDeps, getBackendPathInSourceDir(g, source))
			continue
		}

		buildWrapper, buildWrapperDeps := l.Properties.Build.getBuildWrapperAndDeps(ctx)
		args["build_wrapper"] = buildWrapper

		var sourceWithoutPrefix string
		if buildDir := g.buildDir(); strings.HasPrefix(source, buildDir) {
			sourceWithoutPrefix = source[len(buildDir):]
		} else {
			sourceWithoutPrefix = source
			source = getBackendPathInSourceDir(g, source)
		}
		output := l.ObjDir() + sourceWithoutPrefix + ".o"

		ctx.Build(pctx,
			blueprint.BuildParams{
				Rule:      rule,
				Outputs:   []string{output},
				Inputs:    []string{source},
				Args:      args,
				OrderOnly: utils.NewStringSlice(orderOnly, buildWrapperDeps),
				Optional:  true,
			})
		objectFiles = append(objectFiles, output)
	}

	return objectFiles, nonCompiledDeps
}

// Returns all the source files for a C/C++ library. This includes any sources that are generated.
func (l *library) GetSrcs(ctx blueprint.ModuleContext) []string {
	srcs := l.Properties.getSources(ctx)
	srcs = append(srcs, l.Properties.Build.SourceProps.Specials...)

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == generatedSourceTag },
		func(m blueprint.Module) {
			if gs, ok := m.(dependentInterface); ok {
				srcs = append(srcs, getSourcesGenerated(gs)...)
			} else {
				panic(errors.New(ctx.OtherModuleName(m) + " does not have outputs"))
			}
		})
	return srcs
}

// Returns the whole static dependencies for a library.
func (l *library) GetWholeStaticLibs(ctx blueprint.ModuleContext) []string {
	libs := []string{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == wholeStaticDepTag },
		func(m blueprint.Module) {
			if sl, ok := m.(*staticLibrary); ok {
				libs = append(libs, sl.outputs()...)
			} else if sl, ok := m.(*generateStaticLibrary); ok {
				libs = append(libs, sl.outputs()...)
			} else if _, ok := m.(*externalLib); ok {
				panic(errors.New(ctx.OtherModuleName(m) +
					" is external, so cannot be used in whole_static_libs"))
			} else {
				panic(errors.New(ctx.OtherModuleName(m) + " is not a static library"))
			}
		})

	return libs
}

// Returns all the static library dependencies for a module.
func (l *library) GetStaticLibs(ctx blueprint.ModuleContext) []string {
	libs := []string{}
	for _, moduleName := range l.Properties.ResolvedStaticLibs {
		dep, _ := ctx.GetDirectDep(moduleName)
		if dep == nil {
			panic(fmt.Errorf("%s has no dependency on static lib %s", l.Name(), moduleName))
		}
		if sl, ok := dep.(*staticLibrary); ok {
			libs = append(libs, sl.outputs()...)
		} else if sl, ok := dep.(*generateStaticLibrary); ok {
			libs = append(libs, sl.outputs()...)
		} else if _, ok := dep.(*externalLib); ok {
			// External static libraries are added to the link using the flags
			// exported by their ldlibs and ldflags properties, rather than by
			// specifying the filename here.
		} else {
			panic(errors.New(ctx.OtherModuleName(dep) + " is not a static library"))
		}
	}

	return libs
}

func (g *linuxGenerator) staticLibOutputDir(m *staticLibrary) string {
	return filepath.Join("${BuildDir}", string(m.Properties.TargetType), "static")
}

// The rule for building a static library
// Note that we need to remove the old library, else we will not remove the old object files
var staticLibraryRule = pctx.StaticRule("static_library",
	blueprint.RuleParams{
		Command:     "rm -f $out && $build_wrapper $ar -rcs $out $in",
		Description: "$out",
	}, "ar", "build_wrapper")

var _ = pctx.StaticVariable("whole_static_tool", "${BobScriptsDir}/whole_static.py")
var wholeStaticLibraryRule = pctx.StaticRule("whole_static_library",
	blueprint.RuleParams{
		Command:     "$whole_static_tool --build-wrapper \"$build_wrapper\" --ar $ar --out $out $in $whole_static_libs",
		CommandDeps: []string{"$whole_static_tool"},
		Description: "$out",
	}, "ar", "build_wrapper", "whole_static_libs")

func (g *linuxGenerator) staticActions(m *staticLibrary, ctx blueprint.ModuleContext) {

	// Calculate and record outputs
	m.outputdir = g.staticLibOutputDir(m)
	m.outs = []string{filepath.Join(m.outputDir(), m.outputFileName())}

	rule := staticLibraryRule

	buildWrapper, buildWrapperDeps := m.Properties.Build.getBuildWrapperAndDeps(ctx)

	tc := g.getToolchain(m.Properties.TargetType)
	arBinary, _ := tc.getArchiver()

	args := map[string]string{
		"ar":            arBinary,
		"build_wrapper": buildWrapper,
	}

	wholeStaticLibs := m.library.GetWholeStaticLibs(ctx)
	implicits := wholeStaticLibs

	if len(wholeStaticLibs) > 0 {
		rule = wholeStaticLibraryRule
		args["whole_static_libs"] = strings.Join(wholeStaticLibs, " ")
	}

	// The archiver rules do not allow adding arguments that the user can
	// set, so does not support nonCompiledDeps
	objectFiles, _ := m.library.CompileObjs(ctx)

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      rule,
			Outputs:   m.outputs(),
			Inputs:    objectFiles,
			Implicits: implicits,
			OrderOnly: buildWrapperDeps,
			Optional:  true,
			Args:      args,
		})

	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

// This section contains functions that are common for shared libraries and executables.

func (l *library) getSharedLibLinkPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == sharedDepTag },
		func(m blueprint.Module) {
			if t, ok := m.(targetableModule); ok {
				libs = append(libs, getSharedLibLinkPath(t))
			} else if _, ok := m.(*externalLib); ok {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else {
				panic(errors.New(ctx.OtherModuleName(m) + " doesn't support targets"))
			}
		})
	return
}

func (l *library) getSharedLibFlags(ctx blueprint.ModuleContext) (ldlibs []string, ldflags []string) {
	// With forwarding shared library we do not have to use
	// --no-as-needed for dependencies because it is already set
	useNoAsNeeded := !l.build().isForwardingSharedLibrary()
	hasForwardingLib := false
	libPaths := []string{}
	tc := getBackend(ctx).getToolchain(l.Properties.TargetType)

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == sharedDepTag },
		func(m blueprint.Module) {
			if sl, ok := m.(*sharedLibrary); ok {
				b := sl.build()
				if b.isForwardingSharedLibrary() {
					hasForwardingLib = true
					ldlibs = append(ldlibs, tc.getLinker().keepSharedLibraryTransitivity())
					if useNoAsNeeded {
						ldlibs = append(ldlibs, tc.getLinker().keepUnusedDependencies())
					}
				}
				ldlibs = append(ldlibs, pathToLibFlag(sl.outputName()))
				if b.isForwardingSharedLibrary() {
					if useNoAsNeeded {
						ldlibs = append(ldlibs, tc.getLinker().dropUnusedDependencies())
					}
					ldlibs = append(ldlibs, tc.getLinker().dropSharedLibraryTransitivity())
				}
				if installPath, ok := sl.Properties.InstallableProps.getInstallPath(); ok {
					libPaths = utils.AppendIfUnique(libPaths, installPath)
				}
			} else if sl, ok := m.(*generateSharedLibrary); ok {
				ldlibs = append(ldlibs, pathToLibFlag(sl.outputName()))
				if installPath, ok := sl.generateCommon.Properties.InstallableProps.getInstallPath(); ok {
					libPaths = utils.AppendIfUnique(libPaths, installPath)
				}
			} else if el, ok := m.(*externalLib); ok {
				ldlibs = append(ldlibs, el.exportLdlibs()...)
				ldflags = append(ldflags, el.exportLdflags()...)
			} else {
				panic(errors.New(ctx.OtherModuleName(m) + " is not a shared library"))
			}
		})

	if hasForwardingLib {
		ldlibs = append(ldlibs, tc.getLinker().getForwardingLibFlags())
	}
	if l.Properties.isRpathWanted() {
		if installPath, ok := l.Properties.InstallableProps.getInstallPath(); ok {
			var rpaths []string
			for _, path := range libPaths {
				out, err := filepath.Rel(installPath, path)
				if err != nil {
					panic(fmt.Errorf("Could not find relative path for: %s due to: %s", path, err))
				}
				rpaths = append(rpaths, "'$$ORIGIN/"+out+"'")
			}
			ldlibs = append(ldlibs, tc.getLinker().setRpath(rpaths))
		}
	}
	return
}

func (l *library) getSharedLibraryDir() string {
	return filepath.Join("${BuildDir}", string(l.Properties.TargetType), "shared")
}

func (g *linuxGenerator) sharedLibOutputDir(m *sharedLibrary) string {
	return m.library.getSharedLibraryDir()
}

func (g *linuxGenerator) sharedLibsDir(tgt tgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "shared")
}

func (l *library) getCommonLibArgs(ctx blueprint.ModuleContext) map[string]string {
	ldflags := l.Properties.Ldflags
	tc := getBackend(ctx).getToolchain(l.Properties.TargetType)

	if l.build().isForwardingSharedLibrary() {
		ldflags = append(ldflags, tc.getLinker().keepUnusedDependencies())
	} else {
		ldflags = append(ldflags, tc.getLinker().dropUnusedDependencies())
	}

	versionScript := l.getVersionScript(ctx)
	if versionScript != nil {
		ldflags = append(ldflags, tc.getLinker().setVersionScript(*versionScript))
	}

	sharedLibLdlibs, sharedLibLdflags := l.getSharedLibFlags(ctx)

	linker := tc.getLinker().getTool()
	tcLdflags := tc.getLinker().getFlags()
	tcLdlibs := tc.getLinker().getLibs()
	buildWrapper, _ := l.Properties.Build.getBuildWrapperAndDeps(ctx)

	wholeStaticLibs := l.GetWholeStaticLibs(ctx)
	staticLibs := l.GetStaticLibs(ctx)
	staticLibFlags := []string{}
	if len(wholeStaticLibs) > 0 {
		staticLibFlags = append(staticLibFlags, tc.getLinker().linkWholeArchives(
			wholeStaticLibs))
	}
	staticLibFlags = append(staticLibFlags, staticLibs...)

	args := map[string]string{
		"build_wrapper":   buildWrapper,
		"ldflags":         utils.Join(tcLdflags, ldflags, sharedLibLdflags),
		"linker":          linker,
		"shared_libs_dir": l.getSharedLibraryDir(),
		"shared_libs_flags": utils.Join(append(sharedLibLdlibs,
			tc.getLinker().setRpathLink(l.getSharedLibraryDir()))),
		"static_libs": utils.Join(staticLibFlags),
		"ldlibs":      utils.Join(l.Properties.Ldlibs, tcLdlibs),
	}
	return args
}

func (l *sharedLibrary) getLibArgs(ctx blueprint.ModuleContext) map[string]string {
	args := l.getCommonLibArgs(ctx)
	ldflags := []string{}

	if l.Properties.Library_version != "" {
		var sonameFlag = "-Wl,-soname," + l.getSoname()
		ldflags = append(ldflags, sonameFlag)
	}

	args["ldflags"] += " " + strings.Join(ldflags, " ")

	return args
}

func (b *binary) getLibArgs(ctx blueprint.ModuleContext) map[string]string {
	return b.getCommonLibArgs(ctx)
}

// Returns the implicit dependencies for a library
func (l *library) Implicits(ctx blueprint.ModuleContext) []string {
	implicits := utils.NewStringSlice(l.GetWholeStaticLibs(ctx), l.GetStaticLibs(ctx))
	implicits = append(implicits, l.getSharedLibLinkPaths(ctx)...)
	versionScript := l.getVersionScript(ctx)
	if versionScript != nil {
		implicits = append(implicits, *versionScript)
	}
	return implicits
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

func (g *linuxGenerator) addSharedLibToc(ctx blueprint.ModuleContext, soFile string, tgt tgtType) {
	tc := g.getToolchain(tgt)
	tocFile := soFile + ".toc"
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

// Get the size of the link pool, to limit the number of concurrent link jobs,
// as these are often memory-intensive. This can be overridden with an
// environment variable.
func getLinkParallelism() int {
	if str, ok := os.LookupEnv("BOB_LINK_PARALLELISM"); ok {
		if p, err := strconv.Atoi(str); err == nil {
			return p
		}
	}
	return (runtime.NumCPU() / 5) + 1
}

var linkPoolParams = blueprint.PoolParams{
	Comment: "Limit the parallelization of linking, which is memory intensive",
	Depth:   getLinkParallelism(),
}

var linkPool = pctx.StaticPool("link", linkPoolParams)

var sharedLibraryRule = pctx.StaticRule("shared_library",
	blueprint.RuleParams{
		Command: "$build_wrapper $linker -shared $in -o $out $ldflags " +
			"$static_libs -L$shared_libs_dir $shared_libs_flags $ldlibs",
		Description: "$out",
		Pool:        linkPool,
	}, "build_wrapper", "ldflags", "ldlibs", "linker", "shared_libs_dir", "shared_libs_flags",
	"static_libs")

var symlinkRule = pctx.StaticRule("symlink",
	blueprint.RuleParams{
		Command:     "for i in $out; do ln -nsf $target $$i; done;",
		Description: "$out",
	}, "target")

func (g *linuxGenerator) sharedActions(m *sharedLibrary, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	m.outputdir = g.sharedLibOutputDir(m)
	var soFile string
	if m.library.Properties.Library_version == "" {
		soFile = filepath.Join(m.outputDir(), m.outputName()+m.fileNameExtension)
	} else {
		soFile = filepath.Join(m.outputDir(), m.getRealName())
	}
	m.outs = []string{soFile}

	objectFiles, nonCompiledDeps := m.CompileObjs(ctx)

	_, buildWrapperDeps := m.Properties.Build.getBuildWrapperAndDeps(ctx)

	installDeps := g.install(m, ctx)

	// Create symlinks if needed
	for name, symlinkTgt := range m.librarySymlinks(ctx) {
		symlink := filepath.Join(m.outputDir(), name)
		lib := filepath.Join(m.outputDir(), symlinkTgt)
		ctx.Build(pctx,
			blueprint.BuildParams{
				Rule:     symlinkRule,
				Inputs:   []string{lib},
				Outputs:  []string{symlink},
				Args:     map[string]string{"target": symlinkTgt},
				Optional: true,
			})
		installDeps = append(installDeps, symlink)
	}

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      sharedLibraryRule,
			Outputs:   m.outputs(),
			Inputs:    objectFiles,
			Implicits: append(m.library.Implicits(ctx), nonCompiledDeps...),
			OrderOnly: buildWrapperDeps,
			Optional:  true,
			Args:      m.getLibArgs(ctx),
		})

	g.addSharedLibToc(ctx, soFile, m.getTarget())

	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) binaryOutputDir(m *binary) string {
	return filepath.Join("${BuildDir}", string(m.Properties.TargetType), "executable")
}

var executableRule = pctx.StaticRule("executable",
	blueprint.RuleParams{
		Command: "$build_wrapper $linker $in -o $out $ldflags $static_libs " +
			"-L$shared_libs_dir $shared_libs_flags $ldlibs",
		Description: "$out",
		Pool:        linkPool,
	}, "build_wrapper", "ldflags", "ldlibs", "linker", "shared_libs_dir",
	"shared_libs_flags", "static_libs")

func (g *linuxGenerator) binaryActions(m *binary, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	m.outputdir = g.binaryOutputDir(m)
	m.outs = []string{filepath.Join(m.outputDir(), m.outputName())}

	objectFiles, nonCompiledDeps := m.CompileObjs(ctx)
	/* By default, build all target binaries */
	optional := !isBuiltByDefault(m)

	_, buildWrapperDeps := m.Properties.Build.getBuildWrapperAndDeps(ctx)

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      executableRule,
			Outputs:   m.outputs(),
			Inputs:    objectFiles,
			Implicits: append(m.library.Implicits(ctx), nonCompiledDeps...),
			OrderOnly: buildWrapperDeps,
			Optional:  true,
			Args:      m.getLibArgs(ctx),
		})
	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, optional)
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
