package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/tag"
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

func (g *linuxGenerator) ObjDir(m Compilable) string {
	return filepath.Join("${BuildDir}", string(m.getTarget()), "objects", m.outputName()) + string(os.PathSeparator)
}

type Compilable interface {
	flag.Consumer // Modules which are compilable need to support flags
	flag.Provider // Required for AOSP backend to check for exported flags

	file.Consumer // Compilable objects must match the file consumer interface
	targetableModule
	phonyInterface
	BackendConfigurationProvider
}

// This function has common support to compile objs for static libs, shared libs and binaries.
func (g *linuxGenerator) CompileObjs(l Compilable, ctx blueprint.ModuleContext, tc toolchain.Toolchain) ([]string, []string) {
	orderOnly := GetGeneratedHeadersFiles(ctx)

	as, astargetflags := tc.GetAssembler()
	cc, cctargetflags := tc.GetCCompiler()
	cxx, cxxtargetflags := tc.GetCXXCompiler()

	cflagsList := []string{}
	asflagsList := []string{}
	ccflagsList := []string{}
	cxxflagsList := []string{}

	// Get all the required flags and group them into includes and everything else.
	// This should make it easier to visually inspect the flags in logs/ninja files.
	l.FlagsInTransitive(ctx).GroupByType(flag.TypeInclude).ForEach(
		func(f flag.Flag) {
			switch {
			case (f.Type() & flag.TypeCompilable) == flag.TypeC: //c exclusive flags
				ccflagsList = append(ccflagsList, f.ToString())
			case f.MatchesType(flag.TypeCC | flag.TypeInclude):
				cflagsList = append(cflagsList, f.ToString())
			case f.MatchesType(flag.TypeAsm):
				asflagsList = append(asflagsList, f.ToString())
			case f.MatchesType(flag.TypeCpp):
				cxxflagsList = append(cxxflagsList, strings.Clone(f.ToString()))
			}
		},
	)

	ctx.Variable(pctx, "asflags", utils.Join(astargetflags, asflagsList))
	ctx.Variable(pctx, "cflags", strings.Join(cflagsList, " "))
	ctx.Variable(pctx, "conlyflags", utils.Join(cctargetflags, ccflagsList))
	ctx.Variable(pctx, "cxxflags", utils.Join(cxxtargetflags, cxxflagsList))

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

// Returns the whole static dependencies for a library.
func GetWholeStaticLibs(ctx blueprint.ModuleContext) []string {
	libs := []string{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.WholeStaticTag },
		func(m blueprint.Module) {
			if provider, ok := m.(file.Provider); ok {
				libs = append(libs, provider.OutFiles().ToStringSliceIf(
					func(p file.Path) bool { return p.IsType(file.TypeArchive) },
					func(p file.Path) string { return p.BuildPath() })...)
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
		if provider, ok := dep.(file.Provider); ok {
			libs = append(libs, provider.OutFiles().ToStringSliceIf(
				func(p file.Path) bool { return p.IsType(file.TypeArchive) },
				func(p file.Path) string { return p.BuildPath() })...)
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

type Archivable interface {
	enableable         // For build by default
	dependentInterface // For phony targets
	flag.Consumer      // Modules which are compilable need to support flags
	file.Consumer      // Compilable objects must match the file consumer interface
	BackendConfigurationProvider
}

func (g *linuxGenerator) ArchivableActions(ctx blueprint.ModuleContext,
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

func (g *linuxGenerator) staticActions(m *ModuleStaticLibrary, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	// The archiver rules do not allow adding arguments that the user can
	// set, so does not support nonCompiledDeps
	objectFiles, _ := g.CompileObjs(m, ctx, tc)

	g.ArchivableActions(ctx, m, tc, objectFiles)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))

}

func (g *linuxGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objs, implicits := g.CompileObjs(m, ctx, tc)

	g.SharedLinkActions(ctx, m, tc, objs, implicits)
	g.SharedTocActions(ctx, m)

	g.ArchivableActions(ctx, m, tc, objs)

	installDeps := append(g.install(m, ctx), file.GetOutputs(m)...)
	installDeps = append(installDeps, g.SharedSymlinkActions(ctx, m)...)

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
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.SharedTag },
		func(m blueprint.Module) {
			if e, ok := m.(externableLibrary); ok && e.isExternal() {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else if t, ok := m.(targetableModule); ok {
				libs = append(libs, g.getSharedLibLinkPath(t))
			} else {
				utils.Die("%s doesn't support targets", ctx.OtherModuleName(m))
			}
		})
	return
}

func (g *linuxGenerator) getSharedLibTocPaths(ctx blueprint.ModuleContext) (libs []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.SharedTag },
		func(m blueprint.Module) {
			if e, ok := m.(externableLibrary); ok && e.isExternal() {
				// Don't try and guess the path to external libraries,
				// and as they are outside of the build we don't need to
				// add a dependency on them anyway.
			} else if _, ok := m.(sharedLibProducer); ok { //Remove this check and replace it with an API call
				if m, ok := m.(file.Provider); ok {
					if toc, ok := m.OutFiles().FindSingle(
						func(p file.Path) bool { return p.IsType(file.TypeToc) }); ok {
						libs = append(libs, toc.BuildPath())
					}
				}
			} else {
				utils.Die("%s doesn't produce a shared library", ctx.OtherModuleName(m))
			}
		})
	return
}

func (g *linuxGenerator) getSharedLibFlags(m BackendCommonLibraryInterface, ctx blueprint.ModuleContext) (ldlibs []string, ldflags []string) {
	// With forwarding shared library we do not have to use
	// --no-as-needed for dependencies because it is already set
	useNoAsNeeded := !m.IsForwardingSharedLibrary()
	hasForwardingLib := false
	libPaths := []string{}
	tc := backend.Get().GetToolchain(m.getTarget())

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.SharedTag },
		func(m blueprint.Module) {

			if ex, ok := m.(externableLibrary); ok && ex.isExternal() {
				ldlibs = append(ldlibs, ex.FlagsOut().Filtered(func(f flag.Flag) bool {
					return f.MatchesType(flag.TypeLinkLibrary)
				}).ToStringSlice()...)

				ldflags = append(ldflags, ex.FlagsOut().Filtered(func(f flag.Flag) bool {
					return f.MatchesType(flag.TypeLinker)
				}).ToStringSlice()...)
			} else if sl, ok := m.(*ModuleSharedLibrary); ok {
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
	return
}

// Temporary interface to make library handlers generic between legacy and strict libraries
type BackendCommonLibraryInterface interface {
	flag.Consumer
	targetableModule
	linkableModule
	installable

	// Legacy functions which need a better interface
	IsForwardingSharedLibrary() bool
	IsRpathWanted() bool
	BackendConfigurationProvider
}

type BackendCommonSharedLibraryInterface interface {
	BackendCommonLibraryInterface
	getRealName() string
}

func (g *linuxGenerator) getCommonLibArgs(m BackendCommonLibraryInterface, ctx blueprint.ModuleContext) map[string]string {
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

func (g *linuxGenerator) SharedLinkActions(ctx blueprint.ModuleContext,
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

func (g *linuxGenerator) SharedTocActions(ctx blueprint.ModuleContext,
	m BackendCommonSharedLibraryInterface) {
	if toc, ok := m.OutFiles().FindSingle(
		func(p file.Path) bool { return p.IsType(file.TypeToc) }); ok {
		outputdir := backend.Get().SharedLibsDir(m.getTarget())
		soFile := filepath.Join(outputdir, m.getRealName())
		g.addSharedLibToc(ctx, soFile, toc.BuildPath(), m.getTarget())
	}
}

func (g *linuxGenerator) SharedSymlinkActions(ctx blueprint.ModuleContext,
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

func (g *linuxGenerator) sharedActions(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.getTarget())
	objs, implicits := g.CompileObjs(m, ctx, tc)

	installDeps := g.install(m, ctx)
	g.SharedSymlinkActions(ctx, m)
	g.SharedLinkActions(ctx, m, tc, objs, implicits)
	g.SharedTocActions(ctx, m)

	installDeps = append(installDeps, file.GetOutputs(m)...)
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

func (g *linuxGenerator) strictBinaryActions(m *ModuleStrictBinary, ctx blueprint.ModuleContext) {
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

func (g *linuxGenerator) executableTestActions(m *ModuleTest, ctx blueprint.ModuleContext) {
	// Duplicate behaviour of `bob_executable`
	g.strictBinaryActions(&m.ModuleStrictBinary, ctx)
}
