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
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/toolchain"
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
		Command:     "$build_wrapper $ccompiler -c $cflags $conlyflags -MD -MF $depfile $in -o $out",
		Description: "$out",
	}, "ccompiler", "cflags", "conlyflags", "build_wrapper", "depfile")

var cxxRule = pctx.StaticRule("cxx",
	blueprint.RuleParams{
		Depfile:     "$out.d",
		Deps:        blueprint.DepsGCC,
		Command:     "$build_wrapper $cxxcompiler -c $cflags $cxxflags -MD -MF $depfile $in -o $out",
		Description: "$out",
	}, "cxxcompiler", "cflags", "cxxflags", "build_wrapper", "depfile")

func (m *ModuleLibrary) ObjDir() string {
	return filepath.Join("${BuildDir}", string(m.Properties.TargetType), "objects", m.outputName()) + string(os.PathSeparator)
}

type Compilable interface {
	flag.Consumer // Modules which are compilable need to support flags
	FileConsumer  // Compilable objects must match the file consumer interface

	// Until this can be removed, compilable objects also must support the generated headers interface
	GetGeneratedHeaders(blueprint.ModuleContext) ([]string, []string)

	// Output directory for object files
	ObjDir() string

	GetBuildWrapperAndDeps(blueprint.ModuleContext) (string, []string)
}

// This function has common support to compile objs for static libs, shared libs and binaries.
func CompileObjs(l Compilable, ctx blueprint.ModuleContext, tc toolchain.Toolchain) ([]string, []string) {
	_, orderOnly := l.GetGeneratedHeaders(ctx)

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

			buildWrapper, buildWrapperDeps := l.GetBuildWrapperAndDeps(ctx)
			args["build_wrapper"] = buildWrapper

			output := l.ObjDir() + source.RelBuildPath() + ".o"

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

// Returns the whole static dependencies for a library.
func GetWholeStaticLibs(ctx blueprint.ModuleContext) []string {
	libs := []string{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == WholeStaticTag },
		func(m blueprint.Module) {
			if provider, ok := m.(FileProvider); ok {
				provider.OutFiles().ForEachIf(
					func(fp file.Path) bool {
						return fp.IsType(file.TypeArchive)
					},
					func(fp file.Path) bool {
						libs = append(libs, fp.BuildPath())
						return true
					})
			}
		})

	return libs
}

// Returns all the static library dependencies for a module.
func (m *ModuleLibrary) GetStaticLibs(ctx blueprint.ModuleContext) []string {
	libs := []string{}
	for _, moduleName := range m.Properties.ResolvedStaticLibs {
		dep, _ := ctx.GetDirectDep(moduleName)
		if dep == nil {
			utils.Die("%s has no dependency on static lib %s", m.Name(), moduleName)
		}
		if sl, ok := dep.(*ModuleStaticLibrary); ok {
			libs = append(libs, sl.outputs()...)
		} else if sl, ok := dep.(*generateStaticLibrary); ok {
			libs = append(libs, sl.outputs()...)
		} else if _, ok := dep.(*ModuleExternalLibrary); ok {
			// External static libraries are added to the link using the flags
			// exported by their ldlibs and ldflags properties, rather than by
			// specifying the filename here.
		} else if sl, ok := dep.(*ModuleStrictLibrary); ok {
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

func (g *linuxGenerator) staticActions(m *ModuleStaticLibrary, ctx blueprint.ModuleContext) {

	// Calculate and record outputs
	m.outputdir = backend.Get().StaticLibOutputDir(m.Properties.TargetType)
	m.outs = []string{filepath.Join(m.outputDir(), m.outputFileName())}

	rule := staticLibraryRule

	buildWrapper, buildWrapperDeps := m.Properties.Build.GetBuildWrapperAndDeps(ctx)

	tc := backend.Get().GetToolchain(m.Properties.TargetType)
	arBinary, _ := tc.GetArchiver()

	args := map[string]string{
		"ar":            arBinary,
		"build_wrapper": buildWrapper,
	}

	wholeStaticLibs := GetWholeStaticLibs(ctx)
	implicits := wholeStaticLibs

	if len(wholeStaticLibs) > 0 {
		rule = wholeStaticLibraryRule
		args["whole_static_libs"] = strings.Join(wholeStaticLibs, " ")
	}

	// The archiver rules do not allow adding arguments that the user can
	// set, so does not support nonCompiledDeps
	objectFiles, _ := CompileObjs(m, ctx, tc)

	// OSX workaround, see rule for details.
	if len(objectFiles) == 0 && len(wholeStaticLibs) == 0 && getConfig(ctx).Properties.GetBool("osx") {
		rule = emptyStaticLibraryRule
		// To create an empty lib, we require a dummy object file,
		// we use the detected compiler to emit it.
		cc, _ := tc.GetCCompiler()
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
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == SharedTag },
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

func (g *linuxGenerator) getSharedLibTocPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == SharedTag },
		func(m blueprint.Module) {
			if l, ok := m.(sharedLibProducer); ok {
				libs = append(libs, g.getSharedLibTocPath(l))
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

func (m *ModuleLibrary) getSharedLibFlags(ctx blueprint.ModuleContext) (ldlibs []string, ldflags []string) {
	// With forwarding shared library we do not have to use
	// --no-as-needed for dependencies because it is already set
	useNoAsNeeded := !m.Properties.Build.isForwardingSharedLibrary()
	hasForwardingLib := false
	libPaths := []string{}
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == SharedTag },
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
				ldlibs = append(ldlibs, el.exportLdlibs()...)
				ldflags = append(ldflags, el.exportLdflags()...)
			} else if sl, ok := m.(*ModuleStrictLibrary); ok {
				ldlibs = append(ldlibs, pathToLibFlag(sl.Name()+".so"))
			} else {
				utils.Die("%s is not a shared library", ctx.OtherModuleName(m))
			}
		})

	if hasForwardingLib {
		ldlibs = append(ldlibs, tc.GetLinker().GetForwardingLibFlags())
	}
	if m.Properties.isRpathWanted() {
		if installPath, ok := m.Properties.InstallableProps.getInstallPath(); ok {
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
	return
}

func (g *linuxGenerator) getCommonLibArgs(m *ModuleLibrary, ctx blueprint.ModuleContext) map[string]string {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	ldflags := m.Properties.Ldflags

	if m.Properties.Build.isForwardingSharedLibrary() {
		ldflags = append(ldflags, tc.GetLinker().KeepUnusedDependencies())
	} else {
		ldflags = append(ldflags, tc.GetLinker().DropUnusedDependencies())
	}

	versionScript := m.getVersionScript(ctx)
	if versionScript != nil {
		ldflags = append(ldflags, tc.GetLinker().SetVersionScript(*versionScript))
	}

	sharedLibLdlibs, sharedLibLdflags := m.getSharedLibFlags(ctx)

	linker := tc.GetLinker().GetTool()
	tcLdflags := tc.GetLinker().GetFlags()
	tcLdlibs := tc.GetLinker().GetLibs()
	buildWrapper, _ := m.Properties.Build.GetBuildWrapperAndDeps(ctx)

	wholeStaticLibs := GetWholeStaticLibs(ctx)
	staticLibs := m.GetStaticLibs(ctx)
	staticLibFlags := []string{}
	if len(wholeStaticLibs) > 0 {
		staticLibFlags = append(staticLibFlags, tc.GetLinker().LinkWholeArchives(
			wholeStaticLibs))
	}
	staticLibFlags = append(staticLibFlags, staticLibs...)
	sharedLibDir := backend.Get().SharedLibsDir(m.Properties.TargetType)
	args := map[string]string{
		"build_wrapper":   buildWrapper,
		"ldflags":         utils.Join(tcLdflags, ldflags, sharedLibLdflags),
		"linker":          linker,
		"shared_libs_dir": sharedLibDir,
		"shared_libs_flags": utils.Join(append(sharedLibLdlibs,
			tc.GetLinker().SetRpathLink(sharedLibDir))),
		"static_libs": utils.Join(staticLibFlags),
		"ldlibs":      utils.Join(m.Properties.Ldlibs, tcLdlibs),
	}
	return args
}

func (g *linuxGenerator) getSharedLibArgs(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) map[string]string {
	args := g.getCommonLibArgs(&m.ModuleLibrary, ctx)
	ldflags := []string{}

	if m.Properties.Library_version != "" {
		var sonameFlag = "-Wl,-soname," + m.getSoname()
		ldflags = append(ldflags, sonameFlag)
	}

	args["ldflags"] += " " + strings.Join(ldflags, " ")

	return args
}

func (g *linuxGenerator) getBinaryArgs(m *ModuleBinary, ctx blueprint.ModuleContext) map[string]string {
	return g.getCommonLibArgs(&m.ModuleLibrary, ctx)
}

// Returns the implicit dependencies for a library
// When useToc is set, replace shared libraries with their toc files.
func (g *linuxGenerator) ccLinkImplicits(l linkableModule, ctx blueprint.ModuleContext, useToc bool) []string {
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

func (g *linuxGenerator) sharedActions(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	m.outputdir = backend.Get().SharedLibsDir(m.Properties.TargetType)
	soFile := filepath.Join(m.outputDir(), m.getRealName())
	m.outs = []string{soFile}
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objectFiles, nonCompiledDeps := CompileObjs(m, ctx, tc)

	_, buildWrapperDeps := m.Properties.Build.GetBuildWrapperAndDeps(ctx)

	installDeps := g.install(m, ctx)

	// Sort symlinks
	symlinks := m.librarySymlinks(ctx)
	symlinkKeys := make([]string, len(symlinks))
	keys := reflect.ValueOf(symlinks).MapKeys()

	for i, k := range keys {
		symlinkKeys[i] = k.String()
	}

	sort.Strings(symlinkKeys)

	// Create symlinks if needed
	for _, name := range symlinkKeys {
		symlinkTgt := symlinks[name]
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

func (g *linuxGenerator) binaryActions(m *ModuleBinary, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	m.outputdir = g.binaryOutputDir(m.Properties.TargetType)
	m.outs = []string{filepath.Join(m.outputDir(), m.outputName())}
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objectFiles, nonCompiledDeps := CompileObjs(m, ctx, tc)
	/* By default, build all target binaries */
	optional := !isBuiltByDefault(m)

	_, buildWrapperDeps := m.Properties.Build.GetBuildWrapperAndDeps(ctx)

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
