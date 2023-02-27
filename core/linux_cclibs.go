/*
 * Copyright 2018-2023 Arm Limited.
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
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/utils"
)

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

	expSystemIncludes, expLocalSystemIncludes, expLocalIncludes, expIncludes, exportedCflags := l.GetExportedVariables(ctx)
	// There are 2 sets of include dirs - "global" and "local".
	// Local acts on the root source directory.

	// The order we want is  local_include_dirs, export_local_include_dirs,
	//                       include_dirs, export_include_dirs
	localIncludeDirs := utils.NewStringSlice(l.Properties.Local_include_dirs, l.Properties.Export_local_include_dirs,
		l.Properties.Export_local_system_include_dirs)

	// Prefix all local includes with SrcDir
	localIncludeDirs = utils.PrefixDirs(localIncludeDirs, "${SrcDir}")
	expLocalIncludes = utils.PrefixDirs(expLocalIncludes, "${SrcDir}")
	expLocalSystemIncludes = utils.PrefixDirs(expLocalSystemIncludes, "${SrcDir}")

	gendirs, orderOnly := l.GetGeneratedHeaders(ctx)

	includeDirs := append(localIncludeDirs, l.Properties.Include_dirs...)
	includeDirs = append(includeDirs, l.Properties.Export_include_dirs...)
	includeDirs = append(includeDirs, l.Properties.Export_system_include_dirs...)
	includeDirs = append(includeDirs, expLocalIncludes...)
	includeDirs = append(includeDirs, expIncludes...)
	includeDirs = append(includeDirs, gendirs...)
	includeFlags := utils.PrefixAll(includeDirs, "-I")

	includeSystemDirs := append(expLocalSystemIncludes, expSystemIncludes...)
	systemIncludeFlags := utils.PrefixAll(includeSystemDirs, "-isystem ")

	cflagsList := utils.NewStringSlice(l.Properties.Cflags, l.Properties.Export_cflags,
		exportedCflags, systemIncludeFlags, includeFlags)

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
	srcs := l.Properties.getSourcesResolved(ctx)

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == generatedSourceTag },
		func(m blueprint.Module) {
			if gs, ok := m.(dependentInterface); ok {
				srcs = append(srcs, getSourcesGenerated(gs)...)
			} else {
				utils.Die("%s does not have outputs", ctx.OtherModuleName(m))
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
				utils.Die("%s is external, so cannot be used in whole_static_libs", ctx.OtherModuleName(m))
			} else if _, ok := m.(*strictLibrary); ok {
				// TODO: append lib outputs here, or not, since this is whole_static_libs
			} else {
				utils.Die("%s is not a static library", ctx.OtherModuleName(m))
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
			utils.Die("%s has no dependency on static lib %s", l.Name(), moduleName)
		}
		if sl, ok := dep.(*staticLibrary); ok {
			libs = append(libs, sl.outputs()...)
		} else if sl, ok := dep.(*generateStaticLibrary); ok {
			libs = append(libs, sl.outputs()...)
		} else if _, ok := dep.(*externalLib); ok {
			// External static libraries are added to the link using the flags
			// exported by their ldlibs and ldflags properties, rather than by
			// specifying the filename here.
		} else if sl, ok := dep.(*strictLibrary); ok {
			libs = append(libs, sl.Static.outputs()...)
		} else {
			utils.Die("%s is not a static library", ctx.OtherModuleName(dep))
		}
	}

	return libs
}

// The rule for building a static library
// Note that we need to remove the old library, else we will not remove the old object files
var staticLibraryRule = pctx.StaticRule("static_library",
	blueprint.RuleParams{
		Command:     "rm -f $out && $build_wrapper $ar -rcs $out $in",
		Description: "$out",
	}, "ar", "build_wrapper")

// Creates an empty static library, no objects are specified in this case. Required on OSX as
// a workaround to ar failing to create a library without objects. On linux `!<arch>` as the content
// is sufficient, this is not the case on OSX where ld checks the size of the file.
var emptyStaticLibraryRule = pctx.StaticRule("empty_static_library",
	blueprint.RuleParams{
		Command:     "rm -f $out $out.o && echo \"\" | $ccompiler -o $out.o -c -xc - && $build_wrapper $ar -rcs $out $out.o",
		Description: "$out",
	}, "ccompiler", "ar", "build_wrapper")

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

	// OSX workaround, see rule for details.
	if len(objectFiles) == 0 && len(wholeStaticLibs) == 0 && getConfig(ctx).Properties.GetBool("osx") {
		rule = emptyStaticLibraryRule
		// To create an empty lib, we require a dummy object file,
		// we use the detected compiler to emit it.
		cc, _ := tc.getCCompiler()
		args["ccompiler"] = cc
	}

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

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

// This section contains functions that are common for shared libraries and executables.

// Convert a path to a library into a compiler flag.
// This needs to strip any path, file extension, lib prefix, and prepend -l
func pathToLibFlag(path string) string {
	_, base := filepath.Split(path)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	if !strings.HasPrefix(base, "lib") {
		utils.Die("Shared library name must start with 'lib' prefix")
	}
	base = strings.TrimPrefix(base, "lib")
	return "-l" + base
}

func (g *linuxGenerator) getSharedLibLinkPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == sharedDepTag },
		func(m blueprint.Module) {
			if t, ok := m.(targetableModule); ok {
				libs = append(libs, g.getSharedLibLinkPath(t))
			} else if _, ok := m.(*externalLib); ok {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else {
				utils.Die("%s doesn't support targets", ctx.OtherModuleName(m))
			}
		})
	return
}

func (g *linuxGenerator) getSharedLibTocPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == sharedDepTag },
		func(m blueprint.Module) {
			if l, ok := m.(sharedLibProducer); ok {
				libs = append(libs, g.getSharedLibTocPath(l))
			} else if _, ok := m.(*externalLib); ok {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else {
				utils.Die("%s doesn't produce a shared library", ctx.OtherModuleName(m))
			}
		})
	return
}

func (l *library) getSharedLibFlags(ctx blueprint.ModuleContext) (ldlibs []string, ldflags []string) {
	// With forwarding shared library we do not have to use
	// --no-as-needed for dependencies because it is already set
	useNoAsNeeded := !l.Properties.Build.isForwardingSharedLibrary()
	hasForwardingLib := false
	libPaths := []string{}
	tc := getBackend(ctx).getToolchain(l.Properties.TargetType)

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == sharedDepTag },
		func(m blueprint.Module) {
			if sl, ok := m.(*sharedLibrary); ok {
				b := &sl.library.Properties.Build
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
			} else if sl, ok := m.(*strictLibrary); ok {
				ldlibs = append(ldlibs, pathToLibFlag(sl.Name()+".so"))
			} else {
				utils.Die("%s is not a shared library", ctx.OtherModuleName(m))
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
					utils.Die("Could not find relative path for: %s due to: %s", path, err)
				}
				rpaths = append(rpaths, "'$$ORIGIN/"+out+"'")
			}
			ldlibs = append(ldlibs, tc.getLinker().setRpath(rpaths))
		}
	}
	return
}

func (g *linuxGenerator) getCommonLibArgs(l *library, ctx blueprint.ModuleContext) map[string]string {
	tc := g.getToolchain(l.Properties.TargetType)

	ldflags := l.Properties.Ldflags

	if l.Properties.Build.isForwardingSharedLibrary() {
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
	sharedLibDir := g.sharedLibsDir(l.Properties.TargetType)
	args := map[string]string{
		"build_wrapper":   buildWrapper,
		"ldflags":         utils.Join(tcLdflags, ldflags, sharedLibLdflags),
		"linker":          linker,
		"shared_libs_dir": sharedLibDir,
		"shared_libs_flags": utils.Join(append(sharedLibLdlibs,
			tc.getLinker().setRpathLink(sharedLibDir))),
		"static_libs": utils.Join(staticLibFlags),
		"ldlibs":      utils.Join(l.Properties.Ldlibs, tcLdlibs),
	}
	return args
}

func (g *linuxGenerator) getSharedLibArgs(l *sharedLibrary, ctx blueprint.ModuleContext) map[string]string {
	args := g.getCommonLibArgs(&l.library, ctx)
	ldflags := []string{}

	if l.Properties.Library_version != "" {
		var sonameFlag = "-Wl,-soname," + l.getSoname()
		ldflags = append(ldflags, sonameFlag)
	}

	args["ldflags"] += " " + strings.Join(ldflags, " ")

	return args
}

func (g *linuxGenerator) getBinaryArgs(b *binary, ctx blueprint.ModuleContext) map[string]string {
	return g.getCommonLibArgs(&b.library, ctx)
}

// Returns the implicit dependencies for a library
// When useToc is set, replace shared libraries with their toc files.
func (g *linuxGenerator) ccLinkImplicits(l linkableModule, ctx blueprint.ModuleContext, useToc bool) []string {
	implicits := utils.NewStringSlice(l.GetWholeStaticLibs(ctx), l.GetStaticLibs(ctx))
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
	m.outputdir = g.sharedLibsDir(m.Properties.TargetType)
	soFile := filepath.Join(m.outputDir(), m.getRealName())
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

	orderOnly := buildWrapperDeps
	if enableToc {
		// Add an order only dependecy on the actual libraries to cover
		// the case where the .so is deleted but the toc is still
		// present.
		orderOnly = append(orderOnly, g.getSharedLibLinkPaths(ctx)...)
	}

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      sharedLibraryRule,
			Outputs:   m.outputs(),
			Inputs:    objectFiles,
			Implicits: append(g.ccLinkImplicits(m, ctx, enableToc), nonCompiledDeps...),
			OrderOnly: orderOnly,
			Optional:  true,
			Args:      g.getSharedLibArgs(m, ctx),
		})

	tocFile := g.getSharedLibTocPath(m)
	g.addSharedLibToc(ctx, soFile, tocFile, m.getTarget())

	installDeps = append(installDeps, g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
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
	m.outputdir = g.binaryOutputDir(m.Properties.TargetType)
	m.outs = []string{filepath.Join(m.outputDir(), m.outputName())}

	objectFiles, nonCompiledDeps := m.CompileObjs(ctx)
	/* By default, build all target binaries */
	optional := !isBuiltByDefault(m)

	_, buildWrapperDeps := m.Properties.Build.getBuildWrapperAndDeps(ctx)

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
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      executableRule,
			Outputs:   m.outputs(),
			Inputs:    objectFiles,
			Implicits: append(g.ccLinkImplicits(m, ctx, enableToc), nonCompiledDeps...),
			OrderOnly: orderOnly,
			Optional:  true,
			Args:      g.getBinaryArgs(m, ctx),
		})

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, optional)
}
