/*
 * Copyright 2018-2019 Arm Limited.
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
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/graph"
	"github.com/ARM-software/bob-build/utils"
)

// BuildProps contains properties required by all modules that compile C/C++
type BuildProps struct {
	SourceProps
	AliasableProps

	// Alternate output name, used for the file name and Android rules
	Out string

	// Flags used for C/C++ compilation
	Cflags []string
	// Flags exported for dependent modules
	Export_cflags []string
	// Flags used for C compilation
	Conlyflags []string
	// Flags used for C++ compilation
	Cxxflags []string
	// Flags used for assembly compilation
	Asflags []string
	// Flags used for linking
	Ldflags []string
	// Same as ldflags, but specified on static libraries and propagated to
	// the top-level build object.
	Export_ldflags []string
	// Shared library version
	Library_version string

	// The list of shared lib modules that this library depends on.
	// 'shared_libs' only makes sense for share libraries and executables -
	// static libs should use 'export_shared_libs', so the dependency is
	// propagated to the top-level object.
	Shared_libs []string
	// The libraries mentioned here will be appended to shared_libs of the modules that use
	// this library (via static_libs, whole_static_libs or shared_libs).
	// export_shared_libs is an indication that this module is using a shared library, and
	// users of this module need to link against it.
	Export_shared_libs []string
	ExtraSharedLibs    []string `blueprint:"mutated"`

	// The list of static lib modules that this library depends on
	// Only makes sense to define for shared libraries and executables
	Static_libs []string
	// The libraries mentioned here will be appended to static_libs of the modules that use
	// this library (via static_libs, whole_static_libs or shared_libs).
	// export_static_libs is an indication that this module is using a static library, and
	// users of this module need to link it.
	Export_static_libs []string

	// This list of dependencies that exported cflags and exported include dirs
	// should be propagated 1-level higher
	Reexport_libs []string
	// Internal property for collecting libraries with reexported flags and include paths
	ResolvedReexportedLibs []string `blueprint:"mutated"`

	ResolvedStaticLibs []string `blueprint:"mutated"`

	// The list of whole static libraries that this library depnds on
	// This will include all the objects in the library (as opposed to normal static linking)
	// If this is set for a static library, any shared library will also include objects
	// from dependent libraries
	Whole_static_libs []string

	// List of libraries to import headers from, but not link to
	Header_libs []string

	// List of libraries that users of the current library should import
	// headers from, but not link to
	Export_header_libs []string

	// Linker flags required to link to the necessary system libraries
	Ldlibs []string
	// Same as ldlibs, but specified on static libraries and
	// propagated to the top-level build object.
	Export_ldlibs []string

	// The list of modules that generate extra headers for this module
	Generated_headers []string

	// The list of modules that generate extra source files for this module
	Generated_sources []string

	// The list of modules that generate output required by the build wrapper
	Generated_deps []string

	// Values to use on Android for LOCAL_MODULE_TAGS, defining which builds this module is built for
	// TODO: Hide this in Android-specific properties
	Tags []string

	// Value to use on Android for LOCAL_MODULE_OWNER
	// TODO: Hide this in Android-specific properties
	Owner string

	// The list of include dirs to use that is relative to the source directory
	Include_dirs []string

	// The list of include dirs to use that is relative to the build.bp file
	Local_include_dirs []string // These use relative instead of absolute paths

	// Include local dirs to be exported into dependent
	Export_local_include_dirs []string

	// Include dirs (path relative to root) to be exported into dependent
	Export_include_dirs []string // TODO: Hide this in Android-specific properties

	// Wrapper for all build commands (object file compilation *and* linking)
	Build_wrapper *string
	// Files in the source directory that the wrapper depends on.
	Build_wrapper_deps []string

	// This is a shared library that pulls in one or more shared
	// libraries to resolve symbols that the binary needs. This is
	// useful where a named library is the standard library to link
	// against, but the implementation may exist in another
	// library.
	//
	// Only valid on bob_shared_library.
	//
	// Currently we need to link with -Wl,--copy-dt-needed-entries.
	// This makes the binary depend on the implementation library, and
	// requires the BFD linker.
	Forwarding_shlib *bool

	InstallableProps
	EnableableProps
	SplittableProps

	// Linux kernel config options to emulate. These are passed to Kbuild in
	// the 'make' command-line, and set in the source code via EXTRA_CFLAGS
	Kbuild_options []string
	// Kernel modules which this module depends on
	Extra_symbols []string
	// Arguments to pass to kernel make invocation
	Make_args []string
	// Kernel directory location
	Kernel_dir string
	// Compiler prefix for kernel build
	Kernel_compiler string

	TargetType tgtType `blueprint:"mutated"`
}

// A Build represents the whole tree of properties for a 'library' object,
// including its host and target-specific properties
type Build struct {
	BuildProps
	Target TargetSpecific
	Host   TargetSpecific
}

// These function check the boolean pointers - which are only filled if someone sets them
// If not, the default value is returned

func (l *Build) isHostSupported() bool {
	if l.Host_supported == nil {
		return false
	}
	return *l.Host_supported
}

func (l *Build) isTargetSupported() bool {
	if l.Target_supported == nil {
		return true
	}
	return *l.Target_supported
}

func (l *Build) isForwardingSharedLibrary() bool {
	if l.Forwarding_shlib == nil {
		return false
	}
	return *l.Forwarding_shlib
}

func (l *Build) getBuildWrapperAndDeps(ctx blueprint.ModuleContext) (string, []string) {
	if l.Build_wrapper != nil {
		depargs := map[string]string{}
		files := getDependentArgsAndFiles(ctx, depargs)

		// Replace any property usage in buildWrapper
		buildWrapper := *l.Build_wrapper
		for k, v := range depargs {
			buildWrapper = strings.Replace(buildWrapper, "${"+k+"}", v, -1)
		}

		return buildWrapper, utils.NewStringSlice(l.Build_wrapper_deps, files)
	}

	return "", []string{}
}

// Add module paths to srcs, exclude_srcs, local_include_dirs and export_local_include_dirs
func (l *Build) processPaths(ctx blueprint.BaseModuleContext) {
	prefix := ctx.ModuleDir()
	l.SourceProps.processPaths(ctx)
	l.Local_include_dirs = utils.PrefixDirs(l.Local_include_dirs, prefix)
	l.Export_local_include_dirs = utils.PrefixDirs(l.Export_local_include_dirs, prefix)

	// When prefixPaths is called we have collapsed features, but not
	// targets, so we also need to expand paths in host and target
	// specific properties as well.
	l.Host.SourceProps.processPaths(ctx)
	l.Host.Local_include_dirs = utils.PrefixDirs(l.Host.Local_include_dirs, prefix)
	l.Host.Export_local_include_dirs = utils.PrefixDirs(l.Host.Export_local_include_dirs, prefix)

	l.Target.SourceProps.processPaths(ctx)
	l.Target.Local_include_dirs = utils.PrefixDirs(l.Target.Local_include_dirs, prefix)
	l.Target.Export_local_include_dirs = utils.PrefixDirs(l.Target.Export_local_include_dirs, prefix)
}

// library is a base class for modules which are generated from sets of object files
type library struct {
	blueprint.SimpleName
	Properties struct {
		Features
		Build
		// The list of default properties that should prepended to all configuration
		Defaults []string
	}
}

func (l *library) defaults() []string {
	return l.Properties.Defaults
}

func (l *library) build() *Build {
	return &l.Properties.Build
}

func (l *library) topLevelProperties() []interface{} {
	return []interface{}{&l.Properties.Build.BuildProps}
}

func (l *library) features() *Features {
	return &l.Properties.Features
}

func (l *library) getTarget() tgtType {
	return l.Properties.TargetType
}

func (l *library) getInstallableProps() *InstallableProps {
	return &l.Properties.InstallableProps
}

func (l *library) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, installDepTag, sharedDepTag)
}

func (l *library) getEnableableProps() *EnableableProps {
	return &l.Properties.Build.EnableableProps
}

func (l *library) getAliasList() []string {
	return l.Properties.getAliasList()
}

func (l *library) supportedVariants() (tgts []tgtType) {
	if l.Properties.isHostSupported() {
		tgts = append(tgts, tgtTypeHost)
	}
	if l.Properties.isTargetSupported() {
		tgts = append(tgts, tgtTypeTarget)
	}
	return
}

func (l *library) disable() {
	f := false
	l.Properties.Enabled = &f
}

func (l *library) setVariant(tgt tgtType) {
	l.Properties.TargetType = tgt
}

func (l *library) getSplittableProps() *SplittableProps {
	return &l.Properties.SplittableProps
}

func (l *library) outputName() string {
	if len(l.Properties.Out) > 0 {
		return l.Properties.Out
	}
	return l.SimpleName.Name()
}

func (l *library) altName() string {
	return l.outputName()
}

func (l *library) altShortName() string {
	if len(l.supportedVariants()) > 1 {
		return l.altName() + "__" + string(l.Properties.TargetType)
	}
	return l.altName()
}

// Returns the shortname for the output, which is used as a phony target. If it
// can be built for multiple variants, require a '__host' or '__target' suffix to
// disambiguate.
func (l *library) shortName() string {
	if len(l.supportedVariants()) > 1 {
		return l.Name() + "__" + string(l.Properties.TargetType)
	}
	return l.Name()
}

func (l *library) GetGeneratedHeaders(ctx blueprint.ModuleContext) (includeDirs []string, orderOnly []string) {
	g := getBackend(ctx)
	visited := map[string]bool{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			tag := ctx.OtherModuleDependencyTag(m)
			return tag == generatedHeaderTag || tag == staticDepTag || tag == sharedDepTag
		},
		func(m blueprint.Module) {

			if gs, ok := getGenerateCommon(m); ok {
				// VisitDirectDepsIf will visit a module once for each
				// dependency. Only list the headers once.
				if _, ok := visited[m.Name()]; ok {
					return
				}

				includeDirs = append(includeDirs,
					utils.PrefixDirs(gs.Properties.Export_gen_include_dirs,
						g.sourceOutputDir(gs))...)
				// Generated headers are "order-only". That means that a source file does not need to rebuild
				// if a generated header changes, just that it must be built after a generated header.
				// The source file _will_ be rebuilt if it uses the header (since that is registered in the
				// depfile). Note that this means that generated headers cannot change which headers are used
				// (by aliasing another header).
				ds, ok := m.(dependentInterface)
				if !ok {
					panic(errors.New("generated header must have outputs()"))
				}
				generatedHeaders := getHeadersGenerated(g, ds)

				orderOnly = append(orderOnly, generatedHeaders...)
			}
		})
	return
}

func (l *library) GetExportedVariables(ctx blueprint.ModuleContext) (expLocalIncludes, expIncludes, expCflags []string) {
	visited := map[string]bool{}
	ctx.VisitDirectDeps(func(dep blueprint.Module) {

		if ctx.OtherModuleDependencyTag(dep) == wholeStaticDepTag ||
			ctx.OtherModuleDependencyTag(dep) == staticDepTag ||
			ctx.OtherModuleDependencyTag(dep) == sharedDepTag ||
			ctx.OtherModuleDependencyTag(dep) == flagDepTag {

			if _, ok := visited[dep.Name()]; ok {
				// VisitDirectDeps will visit a module once for each
				// dependency. We've already done this module.
				return
			}

			switch lib := dep.(type) {
			case *staticLibrary:
				expLocalIncludes = append(expLocalIncludes, lib.Properties.Export_local_include_dirs...)
				expIncludes = append(expIncludes, lib.Properties.Export_include_dirs...)
				expCflags = append(expCflags, lib.Properties.Export_cflags...)

			case *sharedLibrary:
				expLocalIncludes = append(expLocalIncludes, lib.Properties.Export_local_include_dirs...)
				expIncludes = append(expIncludes, lib.Properties.Export_include_dirs...)
				expCflags = append(expCflags, lib.Properties.Export_cflags...)
			}
		}
	})

	return
}

func (l *library) processPaths(ctx blueprint.BaseModuleContext) {
	l.Properties.Build.processPaths(ctx)
}

func getLibrary(i interface{}) (*library, bool) {
	v := reflect.Indirect(reflect.ValueOf(i))
	field := v.FieldByName("library")
	ok := false
	var l *library
	if field.IsValid() {
		l, ok = field.Addr().Interface().(*library)
	}
	return l, ok
}

type staticLibrary struct {
	library
}

func (m *staticLibrary) outputDir(g generatorBackend) string {
	return g.staticLibOutputDir(m)
}

func (m *staticLibrary) outputs(g generatorBackend) []string {
	return []string{filepath.Join(m.outputDir(g), m.outputName()+".a")}
}

func (m *staticLibrary) filesToInstall(ctx blueprint.ModuleContext) []string {
	return m.outputs(getBackend(ctx))
}

func (l *library) checkField(cond bool, fieldName string) {
	if !cond {
		panic(fmt.Sprintf("%s has field %s set", l.Name(), fieldName))
	}
}

func (m *staticLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).staticActions(m, ctx)
	}
}

type sharedLibrary struct {
	library
}

func (m *sharedLibrary) getLinkName() string {
	return m.outputName() + ".so"
}

func (m *sharedLibrary) getSoname() string {
	name := m.getLinkName()
	if m.library.Properties.Library_version != "" {
		var v = strings.Split(m.library.Properties.Library_version, ".")
		name += "." + v[0]
	}
	return name
}

func (m *sharedLibrary) getRealName() string {
	name := m.getLinkName()
	if m.library.Properties.Library_version != "" {
		name += "." + m.library.Properties.Library_version
	}
	return name
}

func (m *sharedLibrary) outputDir(g generatorBackend) string {
	return g.sharedLibOutputDir(m)
}

func (m *sharedLibrary) outputs(g generatorBackend) []string {
	if m.library.Properties.Library_version == "" {
		return []string{filepath.Join(m.outputDir(g), m.outputName()+".so")}
	}
	return []string{filepath.Join(m.outputDir(g), m.getRealName())}
}

func (m *sharedLibrary) filesToInstall(ctx blueprint.ModuleContext) []string {
	return m.outputs(getBackend(ctx))
}

func (m *sharedLibrary) librarySymlinks(ctx blueprint.ModuleContext) map[string]string {
	symlinks := map[string]string{}

	if m.library.Properties.Library_version != "" {
		// To build you need a symlink from the link name and soname.
		// At runtime only the soname symlink is required.
		soname := m.getSoname()
		realName := m.getRealName()
		if soname == realName {
			panic(fmt.Errorf("module %s has invalid library_version '%s'",
				m.Name(),
				m.library.Properties.Library_version))
		}
		symlinks[m.getLinkName()] = soname
		symlinks[soname] = realName
	}

	return symlinks
}

func (m *sharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).sharedActions(m, ctx)
	}
}

type binary struct {
	library
}

func (m *binary) outputDir(g generatorBackend) string {
	return g.binaryOutputDir(m)
}

func (m *binary) outputs(g generatorBackend) []string {
	return []string{filepath.Join(m.outputDir(g), m.outputName())}
}

func (m *binary) filesToInstall(ctx blueprint.ModuleContext) []string {
	return m.outputs(getBackend(ctx))
}

func (m *binary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).binaryActions(m, ctx)
	}
}

func (l *library) LibraryFactory(config *bobConfig, module blueprint.Module) (blueprint.Module, []interface{}) {
	availableFeatures := config.getAvailableFeatures()
	l.Properties.Features.Init(availableFeatures, BuildProps{})
	l.Properties.Build.Host.Features.Init(availableFeatures, BuildProps{})
	l.Properties.Build.Target.Features.Init(availableFeatures, BuildProps{})

	return module, []interface{}{&l.Properties, &l.SimpleName.Properties}
}

func staticLibraryFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &staticLibrary{}
	return module.LibraryFactory(config, module)
}

func sharedLibraryFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &sharedLibrary{}
	return module.LibraryFactory(config, module)
}

func binaryFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &binary{}
	return module.LibraryFactory(config, module)
}

func getBinaryOrSharedLib(m blueprint.Module) (*library, bool) {
	if sl, ok := m.(*sharedLibrary); ok {
		return &sl.library, true
	} else if b, ok := m.(*binary); ok {
		return &b.library, true
	}

	return nil, false
}

func checkLibraryFieldsMutator(mctx blueprint.BottomUpMutatorContext) {
	m := mctx.Module()
	if b, ok := m.(*binary); ok {
		props := b.Properties
		b.checkField(len(props.Export_cflags) == 0, "export_cflags")
		b.checkField(len(props.Export_include_dirs) == 0, "export_include_dirs")
		b.checkField(len(props.Export_ldflags) == 0, "export_ldflags")
		b.checkField(len(props.Export_local_include_dirs) == 0, "export_local_include_dirs")
		b.checkField(len(props.Export_shared_libs) == 0, "export_shared_libs")
		b.checkField(len(props.Export_static_libs) == 0, "export_static_libs")
		b.checkField(len(props.Reexport_libs) == 0, "reexport_libs")
		b.checkField(len(props.Export_ldlibs) == 0, "export_ldlibs")
		b.checkField(len(props.Whole_static_libs) == 0, "whole_static_libs")
		b.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
	} else if sl, ok := m.(*sharedLibrary); ok {
		props := sl.Properties
		sl.checkField(len(props.Export_ldflags) == 0, "export_ldflags")
		sl.checkField(len(props.Export_shared_libs) == 0, "export_shared_libs")
		sl.checkField(len(props.Export_static_libs) == 0, "export_static_libs")
		sl.checkField(len(props.Export_ldlibs) == 0, "export_ldlibs")
	} else if sl, ok := m.(*staticLibrary); ok {
		props := sl.Properties
		sl.checkField(len(props.Shared_libs) == 0, "shared_libs")
		sl.checkField(len(props.Static_libs) == 0, "static_libs")
		sl.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
	}

}

// Traverse the dependency tree, following all StaticDepTag and WholeStaticDepTag links.
// Do *not* include modules which are in the tree via any other dependency tag.
func getLinkableModules(mctx blueprint.TopDownMutatorContext) map[blueprint.Module]bool {
	ret := make(map[blueprint.Module]bool)

	mctx.WalkDeps(func(dep blueprint.Module, parent blueprint.Module) bool {
		// Stop iteration once we get to other kinds of dependency which won't
		// actually be linked.
		if mctx.OtherModuleDependencyTag(dep) != staticDepTag &&
			mctx.OtherModuleDependencyTag(dep) != wholeStaticDepTag {
			return false
		}
		ret[dep] = true

		return true
	})

	return ret
}

// Check that no libraries are being accidentally linked twice, by having one copy
// linked explicitly (via export_static_libs), and another included in a different
// library via whole_static_libs.
func checkForMultipleLinking(topLevelModuleName string, allExportStaticLibs map[string]bool, insideWholeLibs map[string]string) {
	duplicateDeps := []string{}
	for dep := range allExportStaticLibs {
		if _, ok := insideWholeLibs[dep]; ok {
			duplicateDeps = append(duplicateDeps, dep)
		}
	}

	if len(duplicateDeps) > 0 {
		msg := fmt.Sprintf("Warning: %s links with the following libraries multiple times:\n", topLevelModuleName)
		for _, dep := range duplicateDeps {
			msg += fmt.Sprintf("  * %s, but also %s, which includes %s as a whole_static_lib\n",
				dep, insideWholeLibs[dep], dep)
		}
		panic(msg)
	}
}

// While traversing the static library dependency tree, propagate extra properties.
func propagateOtherExportedProperties(l *library, depLib *staticLibrary) {
	props := l.build()
	for _, shLib := range depLib.Properties.Export_shared_libs {
		if !utils.Contains(props.Shared_libs, shLib) {
			props.Shared_libs = append(props.Shared_libs, shLib)
			props.ExtraSharedLibs = append(props.ExtraSharedLibs, shLib)
		}
	}
	for _, ldlib := range depLib.Properties.Export_ldlibs {
		if !utils.Contains(props.Ldlibs, ldlib) {
			props.Ldlibs = append(props.Ldlibs, ldlib)
		}
	}
	props.Ldflags = append(props.Ldflags, depLib.Properties.Export_ldflags...)

	// Header libraries are *not* propagated here, because they are currently
	// only supported on Android, which will automatically re-export them just
	// by adding them to LOCAL_EXPORT_HEADER_LIBRARY_HEADERS.
}

func exportLibFlagsMutator(mctx blueprint.TopDownMutatorContext) {
	l, ok := getBinaryOrSharedLib(mctx.Module())
	if !ok {
		return
	}

	// Track the set of everything mentioned in 'export_static_libs' of all
	// dependencies of this module, for multiple-link checking.
	allExportStaticLibs := make(map[string]bool)
	// Map between a library name and the first encountered lib in which it
	// is used in whole_static_libs.
	insideWholeLibs := make(map[string]string)
	// VisitDepsDepthFirst doesn't let us stop iteration, so get the list of
	// modules to examine separately using WalkDeps.
	modulesToVisit := getLinkableModules(mctx)

	mctx.VisitDepsDepthFirst(func(dep blueprint.Module) {
		if _, ok := modulesToVisit[dep]; !ok {
			return
		}

		if depLib, ok := dep.(*staticLibrary); ok {
			for _, subLib := range depLib.Properties.Whole_static_libs {
				if firstContainingLib, ok := insideWholeLibs[subLib]; ok {
					panic(fmt.Sprintf("%s links with %s and %s, which both contain %s as whole_static_libs",
						mctx.Module().Name(), firstContainingLib,
						depLib.Name(), subLib))
				} else {
					insideWholeLibs[subLib] = depLib.Name()
				}
			}
			for _, subLib := range depLib.Properties.Export_static_libs {
				allExportStaticLibs[subLib] = true
			}

			propagateOtherExportedProperties(l, depLib)
		} else if _, ok := dep.(*generateStaticLibrary); ok {
			// Nothing to do for GeneratedStaticLibrary
			//
			// The GeneratedStaticLibrary is expected to be self
			// contained, so no pulling in of other static or shared
			// libraries.
		} else if _, ok := dep.(*externalLib); ok {
			// External libary dependencies are not handled.
		} else {
			panic(fmt.Sprintf("%s is not a staticLibrary", dep.Name()))
		}

		// Don't add whole_static_lib components to the library list, because their
		// contents are already included in the parent library.
		if mctx.OtherModuleDependencyTag(dep) != wholeStaticDepTag && mctx.OtherModuleDependencyTag(dep) != staticDepTag {
			panic(fmt.Sprintf("Non WholeStatic or Static dep tag encountered visiting %s from %s",
				dep.Name(), mctx.ModuleName()))
		}
	})

	checkForMultipleLinking(mctx.ModuleName(), allExportStaticLibs, insideWholeLibs)
}

type graphMutatorHandler struct {
	graph graph.Graph
}

const (
	maxInt = int(^uint(0) >> 1)
	minInt = -maxInt - 1
)

func (handler *graphMutatorHandler) ResolveDependencySortMutator(mctx blueprint.BottomUpMutatorContext) {
	mainModule := mctx.Module()
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			return // Not enabled, so not needed
		}
	}
	if _, ok := mainModule.(*defaults); ok {
		return // ignore bob_defaults
	}
	handler.graph.AddNode(mainModule.Name())

	var mainBuild *Build
	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		mainBuild = buildProps.build()
	} else {
		return // ignore not a build
	}

	for _, lib := range mainBuild.Static_libs {
		if _, err := handler.graph.AddEdgeToExistingNodes(mainModule.Name(), lib); err != nil {
			panic(fmt.Errorf("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModule.Name(), lib, lib))
		}
		handler.graph.SetEdgeColor(mainModule.Name(), lib, "blue")
	}

	for _, lib := range mainBuild.Export_static_libs {
		if _, err := handler.graph.AddEdgeToExistingNodes(mainModule.Name(), lib); err != nil {
			panic(fmt.Errorf("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModule.Name(), lib, lib))
		}
		handler.graph.SetEdgeColor(mainModule.Name(), lib, "green")
	}

	for _, lib := range mainBuild.Whole_static_libs {
		if _, err := handler.graph.AddEdgeToExistingNodes(mainModule.Name(), lib); err != nil {
			panic(fmt.Errorf("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModule.Name(), lib, lib))
		}
		handler.graph.SetEdgeColor(mainModule.Name(), lib, "red")
	}

	temporaryPaths := map[string][]string{} // For preserving order in declaration

	for i, previous := range mainBuild.Static_libs {
		for j := i + 1; j < len(mainBuild.Static_libs); j++ {
			lib := mainBuild.Static_libs[j]
			if !handler.graph.IsReachable(lib, previous) {
				if handler.graph.AddEdge(previous, lib) {
					temporaryPaths[previous] = append(temporaryPaths[previous], lib)
					handler.graph.SetEdgeColor(previous, lib, "pink")
				}
			}
		}
	}

	for i, previous := range mainBuild.Export_static_libs {
		for j := i + 1; j < len(mainBuild.Export_static_libs); j++ {
			lib := mainBuild.Export_static_libs[j]
			if !handler.graph.IsReachable(lib, previous) {
				if handler.graph.AddEdge(previous, lib) {
					temporaryPaths[previous] = append(temporaryPaths[previous], lib)
					handler.graph.SetEdgeColor(previous, lib, "pink")
				}
			}
		}
	}

	sub := graph.GetSubgraph(handler.graph, mainModule.Name())

	// Remove temporary path
	for key, list := range temporaryPaths {
		for _, value := range list {
			handler.graph.DeleteEdge(key, value)
		}
	}

	// The order of static libraries influences performance by
	// influencing memory layout. Where possible we want libraries
	// that depend on each other to be as close as possible. Library
	// order is determined by a topological sort.  Setting the
	// priority changes the order that child nodes are visited.
	//
	// Libraries that are frequently called are more
	// important and should be close to their callers. This information is not available in bob,
	// so estimate this with the number of users.
	//
	// Libraries that are large, or will cause a large number of
	// libraries to occur in the middle of the list, should be at the
	// end of the list. Treat this as the cost of visiting the
	// library. As an estimate of cost, count the number of libraries
	// that would be pulled in.
	//
	// The node priority is calculated as 'A * importance - cost',
	// where A is an arbitraty scaling factor.
	//
	// This is a bottom up mutator, so by the time we get to a binary
	// (or shared library), this mutator will have run on all their
	// dependencies and the (shared) graph will be complete (for the
	// current module).
	for _, nodeID := range sub.GetNodes() {
		cost := graph.GetSubgraph(sub, nodeID).GetNodeCount()
		sources, _ := sub.GetSources(nodeID)
		priority := len(sources)
		sub.SetNodePriority(nodeID, (10*priority)-cost)
	}

	// The main library must always be evaluated first in the topological sort
	sub.SetNodePriority(mainModule.Name(), minInt)

	// We want those edges for calculating priority. After setting priority we can remove them.
	sub.DeleteProxyEdges("red")

	sub2 := graph.GetSubgraph(sub, mainModule.Name())
	sortedStaticLibs, isDAG := graph.TopologicalSort(sub2)

	// Pop the module itself from the front of the list
	sortedStaticLibs = sortedStaticLibs[1:]

	if !isDAG {
		panic("We have detected cycle: " + mainModule.Name())
	} else {
		mainBuild.ResolvedStaticLibs = sortedStaticLibs
	}

	alreadyAddedStaticLibsDependencies := utils.NewStringSlice(mainBuild.Static_libs, mainBuild.Export_static_libs)
	extraStaticLibsDependencies := utils.Difference(mainBuild.ResolvedStaticLibs, alreadyAddedStaticLibsDependencies)

	mctx.AddVariationDependencies(nil, staticDepTag, extraStaticLibsDependencies...)

	// This module may now depend on extra shared libraries, inherited from included
	// static libraries. Add that dependency here.
	mctx.AddVariationDependencies(nil, sharedDepTag, mainBuild.ExtraSharedLibs...)
}
