/*
 * Copyright 2018-2021 Arm Limited.
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
	"regexp"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/graph"
	"github.com/ARM-software/bob-build/internal/utils"
)

const (
	tocExt = ".toc"
)

var depOutputsVarRegexp = regexp.MustCompile(`^\$\{(.+)_out\}$`)

type propertyExporter interface {
	exportCflags() []string
	exportIncludeDirs() []string
	exportLdflags() []string
	exportLdlibs() []string
	exportLocalIncludeDirs() []string
	exportSharedLibs() []string
}

// CommonProps defines a set of properties which are common
// for multiple module types.
type CommonProps struct {
	SourceProps
	IncludeDirsProps
	InstallableProps
	EnableableProps
	AndroidProps
	AliasableProps

	// Flags used for C compilation
	Cflags []string
}

// BuildProps contains properties required by all modules that compile C/C++
type BuildProps struct {
	// Alternate output name, used for the file name and Android rules
	Out *string
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
	// Shared library version script
	Version_script *string

	// The list of shared lib modules that this library depends on.
	// These are propagated to the closest linking object when specified on static libraries.
	// shared_libs is an indication that this module is using a shared library, and
	// users of this module need to link against it.
	Shared_libs []string `bob:"first_overrides"`
	// The libraries mentioned here will be appended to shared_libs of the modules that use
	// this library (via static_libs, whole_static_libs or shared_libs).
	ExtraSharedLibs []string `blueprint:"mutated"`

	// The list of static lib modules that this library depends on
	// These are propagated to the closest linking object when specified on static libraries.
	// static_libs is an indication that this module is using a static library, and
	// users of this module need to link against it.
	Static_libs []string `bob:"first_overrides"`

	// This list of dependencies that exported cflags and exported include dirs
	// should be propagated 1-level higher
	Reexport_libs []string `bob:"first_overrides"`
	// Internal property for collecting libraries with reexported flags and include paths
	ResolvedReexportedLibs []string `blueprint:"mutated"`

	ResolvedStaticLibs []string `blueprint:"mutated"`

	// The list of whole static libraries that this library depnds on
	// This will include all the objects in the library (as opposed to normal static linking)
	// If this is set for a static library, any shared library will also include objects
	// from dependent libraries
	Whole_static_libs []string `bob:"first_overrides"`

	// List of libraries to import headers from, but not link to
	Header_libs []string `bob:"first_overrides"`

	// List of libraries that users of the current library should import
	// headers from, but not link to
	Export_header_libs []string `bob:"first_overrides"`

	// Linker flags required to link to the necessary system libraries
	// These are propagated to the closest linking object when specified on static libraries.
	Ldlibs []string `bob:"first_overrides"`

	// The list of modules that generate extra headers for this module
	Generated_headers []string `bob:"first_overrides"`

	// The list of modules that generate extra headers for this module,
	// which should be made available to linking modules
	Export_generated_headers []string `bob:"first_overrides"`

	// The list of modules that generate extra source files for this module
	Generated_sources []string

	// The list of modules that generate output required by the build wrapper
	Generated_deps []string

	// Include local dirs to be exported into dependent
	Export_local_include_dirs []string `bob:"first_overrides"`

	// Include dirs (path relative to root) to be exported into dependent
	Export_include_dirs []string `bob:"first_overrides"`

	// Wrapper for all build commands (object file compilation *and* linking)
	Build_wrapper *string

	// Adds DT_RPATH symbol to binaries and shared libraries so that they can find
	// their dependencies at runtime.
	Add_lib_dirs_to_rpath *bool

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

	StripProps
	AndroidPGOProps

	TargetType tgtType `blueprint:"mutated"`
}

func (b *BuildProps) processBuildWrapper(ctx blueprint.BaseModuleContext) {
	if b.Build_wrapper != nil {
		// The build wrapper may be a local tool, in which case we
		// need to prefix it with ${SrcDir}. It can also be a tool in
		// PATH like ccache.
		//
		// We want to avoid doing this repeatedly, so try do it in an
		// early mutator
		*b.Build_wrapper = strings.TrimSpace(*b.Build_wrapper)
		firstWord := strings.SplitN(*b.Build_wrapper, " ", 1)[0]

		// If the first character is '/' this is an absolute path, so no need to do anything
		if firstWord[0] != '/' {
			// Otherwise if the first word contains '/' this is a local path
			if strings.ContainsAny(firstWord, "/") {
				*b.Build_wrapper = getBackendPathInSourceDir(getBackend(ctx), *b.Build_wrapper)
			}
		}
	}
}

// A Build represents the whole tree of properties for a 'library' object,
// including its host and target-specific properties
type Build struct {
	CommonProps
	BuildProps
	Target TargetSpecific
	Host   TargetSpecific
	SplittableProps
}

func (l *Build) getTargetSpecific(tgt tgtType) *TargetSpecific {
	if tgt == tgtTypeHost {
		return &l.Host
	} else if tgt == tgtTypeTarget {
		return &l.Target
	} else {
		panic(fmt.Errorf("Unsupported target type: %s", tgt))
	}
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

func (l *Build) isRpathWanted() bool {
	if l.Add_lib_dirs_to_rpath == nil {
		return false
	}
	return *l.Add_lib_dirs_to_rpath
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

		return buildWrapper, files
	}

	return "", []string{}
}

// Add module paths to srcs, exclude_srcs, local_include_dirs, export_local_include_dirs
// and post_install_tool
func (l *BuildProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)

	l.Export_local_include_dirs = utils.PrefixDirs(l.Export_local_include_dirs, prefix)
	l.processBuildWrapper(ctx)
}

func (l *Build) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	l.BuildProps.processPaths(ctx, g)
	l.CommonProps.processPaths(ctx, g)
}

func (c *CommonProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)

	c.SourceProps.processPaths(ctx, g)
	c.InstallableProps.processPaths(ctx, g)
	c.IncludeDirsProps.Local_include_dirs = utils.PrefixDirs(c.IncludeDirsProps.Local_include_dirs, prefix)
}

// library is a base class for modules which are generated from sets of object files
type library struct {
	moduleBase
	simpleOutputProducer

	Properties struct {
		Features
		Build
		// The list of default properties that should prepended to all configuration
		Defaults []string

		VersionScriptModule *string `blueprint:"mutated"`
	}
}

// library supports the following functionality:
// * sharing properties via defaults
// * feature-specific properties
// * target-specific properties
// * installation
// * module enabling/disabling
// * exporting properties to other modules
// * use of {{match_srcs}} on some properties
// * properties that require escaping
// * appending to aliases
var _ defaultable = (*library)(nil)
var _ featurable = (*library)(nil)
var _ targetSpecificLibrary = (*library)(nil)
var _ installable = (*library)(nil)
var _ enableable = (*library)(nil)
var _ propertyExporter = (*library)(nil)
var _ matchSourceInterface = (*library)(nil)
var _ propertyEscapeInterface = (*library)(nil)
var _ aliasable = (*library)(nil)

func (l *library) defaults() []string {
	return l.Properties.Defaults
}

func (l *library) defaultableProperties() []interface{} {
	return []interface{}{
		&l.Properties.Build.CommonProps,
		&l.Properties.Build.BuildProps,
		&l.Properties.Build.SplittableProps,
	}
}

func (l *library) build() *Build {
	return &l.Properties.Build
}

func (l *library) featurableProperties() []interface{} {
	return []interface{}{
		&l.Properties.Build.CommonProps,
		&l.Properties.Build.BuildProps,
		&l.Properties.Build.SplittableProps,
	}
}

func (l *library) targetableProperties() []interface{} {
	return []interface{}{
		&l.Properties.Build.CommonProps,
		&l.Properties.Build.BuildProps,
		&l.Properties.Build.SplittableProps,
	}
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

// Return the shortName of dependencies which must be installed alongside the
// library. Exclude external libraries - these will never be added via
// install_deps, but may end up in shared_libs.
func (l *library) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsIf(ctx,
		func(m blueprint.Module) bool {
			tag := ctx.OtherModuleDependencyTag(m)
			// External libraries do not have a build target so don't
			// try to add a dependency on them.
			if _, ok := m.(*externalLib); ok {
				return false
			}
			if tag == installDepTag || tag == sharedDepTag {
				return true
			}
			return false
		})
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

func (l *library) getTargetSpecific(tgt tgtType) *TargetSpecific {
	return l.Properties.getTargetSpecific(tgt)
}

func (l *library) outputName() string {
	if l.Properties.Out != nil {
		return *l.Properties.Out
	}
	return l.Name()
}

func (l *library) getDebugInfo() *string {
	return l.Properties.getDebugInfo()
}

func (l *library) getDebugPath() *string {
	return l.Properties.getDebugPath()
}

func (l *library) setDebugPath(path *string) {
	l.Properties.setDebugPath(path)
}

func (m *library) stripOutputDir(g generatorBackend) string {
	return getBackendPathInBuildDir(g, string(m.Properties.TargetType), "strip")
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

func (l *library) getEscapeProperties() []*[]string {
	return []*[]string{
		&l.Properties.Asflags,
		&l.Properties.Cflags,
		&l.Properties.Conlyflags,
		&l.Properties.Cxxflags,
		&l.Properties.Ldflags}
}

func (l *library) getSourceProperties() *SourceProps {
	return &l.Properties.SourceProps
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `BuildProps` this is:
//  - Ldflags
//  - Cflags
//  - Conlyflags
//  - Cxxflags
func (l *library) getMatchSourcePropNames() []string {
	return []string{"Ldflags", "Cflags", "Conlyflags", "Cxxflags"}
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
	visited := map[string]bool{}

	mainModule := ctx.Module()

	ctx.WalkDeps(func(child, parent blueprint.Module) bool {

		tag := ctx.OtherModuleDependencyTag(child)

		/* We want all the export_gen_include_dirs from generated modules mentioned by the
		 * main module, primarily from generated_headers, but also static_libs and
		 * shared_libs where they refer to a bob_generated_[static|shared]_library.
		 *
		 * We also want all the export_generated_headers from libraries mentioned by the main
		 * module, i.e. from static_libs and shared_libs, as well as
		 * export_generated_headers from the main module itself.
		 *
		 * Any time we visit a generated module, we must also visit its children, in case
		 * they are encapsulated. At this point, the only subsequent modules visited will be
		 * other generated modules via the `encapsulates` property.
		 *
		 * Note that generated_header and export_generated_header tags can't have child
		 * generated_header, export_generated_header, static_libs or shared_libs tags,
		 * because these are only added by libraries.
		 */
		importHeaderDirs := false
		visitChildren := false
		childMustBeGenerated := true
		if parent == mainModule {
			if tag == generatedHeaderTag || tag == exportGeneratedHeaderTag {
				importHeaderDirs = true
				// Check top level generated header modules for encapsulated modules
				visitChildren = true
			} else if tag == staticDepTag || tag == sharedDepTag || tag == reexportLibsTag {
				/* Try to import generated header dirs from static|shared_libs too:
				 * - The library could be a bob_generate_shared_library or
				 *   bob_generate_static_library, in which case we need to import
				 *   any generated header dirs it exports.
				 * - If it's a bob_static_library or bob_shared_library, it may
				 *   export generated header dirs, so it's children need visiting.
				 * In either case we need to keep recursing in case of encapsulated
				 * modules.
				 */
				importHeaderDirs = true
				visitChildren = true
				// We don't know the module type so disable the check
				childMustBeGenerated = false
			}
		} else {
			if tag == exportGeneratedHeaderTag {
				importHeaderDirs = true
				// Visit children of exported gen header dirs
				// in case the module encapsulates anything.
				visitChildren = true
			} else if tag == encapsulatesTag {
				importHeaderDirs = true
				// Keep walking encapsulated modules indefinitely
				visitChildren = true
			}
		}

		if importHeaderDirs {
			// Add include directories for any generated modules
			if gs, ok := getGenerateCommon(child); ok {
				// WalkDeps will visit a module once for each
				// dependency tag. Only list the headers once.
				if _, seen := visited[child.Name()]; !seen {
					visited[child.Name()] = true

					includeDirs = append(includeDirs, gs.genIncludeDirs()...)

					// Generated headers are "order-only". That means that a source file does not need to rebuild
					// if a generated header changes, just that it must be built after a generated header.
					// The source file _will_ be rebuilt if it uses the header (since that is registered in the
					// depfile). Note that this means that generated headers cannot change which headers are used
					// (by aliasing another header).
					ds, ok := child.(dependentInterface)
					if !ok {
						panic(fmt.Errorf("generated_headers %s must have outputs()", child.Name()))
					}

					orderOnly = append(orderOnly, getHeadersGenerated(ds)...)
				}
			} else if childMustBeGenerated {
				panic(fmt.Errorf("%s dependency on non-generated module %s", tag.(dependencyTag).name, child.Name()))
			}
		}

		return visitChildren
	})
	return
}

func (l *library) getAllGeneratedSourceModules(ctx blueprint.ModuleContext) (modules []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == generatedSourceTag },
		func(m blueprint.Module) {
			if gs, ok := getGenerateCommon(m); ok {
				// Add our own name
				modules = append(modules, gs.Name())
				// Add transitively encapsulated module names (if any)
				modules = append(modules, gs.encapsulatedModules()...)
			}
		})
	return
}

func (l *library) GetExportedVariables(ctx blueprint.ModuleContext) (expLocalIncludes, expIncludes, expCflags []string) {
	visited := map[string]bool{}
	ctx.VisitDirectDeps(func(dep blueprint.Module) {

		if !(ctx.OtherModuleDependencyTag(dep) == wholeStaticDepTag ||
			ctx.OtherModuleDependencyTag(dep) == staticDepTag ||
			ctx.OtherModuleDependencyTag(dep) == sharedDepTag ||
			ctx.OtherModuleDependencyTag(dep) == reexportLibsTag) {
			return
		} else if _, ok := visited[dep.Name()]; ok {
			// VisitDirectDeps will visit a module once for each
			// dependency. We've already done this module.
			return
		}
		visited[dep.Name()] = true

		if pe, ok := dep.(propertyExporter); ok {
			expLocalIncludes = append(expLocalIncludes, pe.exportLocalIncludeDirs()...)
			expIncludes = append(expIncludes, pe.exportIncludeDirs()...)
			expCflags = append(expCflags, pe.exportCflags()...)
		}
	})

	return
}

func (l *library) getVersionScript(ctx blueprint.ModuleContext) *string {
	if l.Properties.VersionScriptModule != nil {
		module, _ := ctx.GetDirectDep(*l.Properties.VersionScriptModule)
		outputs := module.(dependentInterface).outputs()
		if len(outputs) != 1 {
			panic(errors.New(ctx.OtherModuleName(module) + " must have exactly one output"))
		}
		return &outputs[0]
	}

	if l.Properties.Build.Version_script != nil {
		path := getBackendPathInSourceDir(getBackend(ctx), *l.Properties.Build.Version_script)
		return &path
	}

	return nil
}

func (l *library) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	l.Properties.Build.processPaths(ctx, g)

	versionScript := l.Properties.Build.Version_script
	if versionScript != nil {
		matches := depOutputsVarRegexp.FindStringSubmatch(*versionScript)
		if len(matches) == 2 {
			l.Properties.VersionScriptModule = &matches[1]
		} else {
			*versionScript = filepath.Join(projectModuleDir(ctx), *versionScript)
		}
	}
}

func (m *library) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	return m.outputs()
}

func (l *library) checkField(cond bool, fieldName string) {
	if !cond {
		panic(fmt.Sprintf("%s has field %s set", l.Name(), fieldName))
	}
}

// All libraries must implement `propertyExporter`
func (l *library) exportCflags() []string           { return l.Properties.Export_cflags }
func (l *library) exportIncludeDirs() []string      { return l.Properties.Export_include_dirs }
func (l *library) exportLocalIncludeDirs() []string { return l.Properties.Export_local_include_dirs }
func (l *library) exportLdflags() []string          { return l.Properties.Export_ldflags }
func (l *library) exportLdlibs() []string           { return l.Properties.Ldlibs }
func (l *library) exportSharedLibs() []string       { return l.Properties.Shared_libs }

type staticLibrary struct {
	library
}

func (m *staticLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).staticActions(m, ctx)
	}
}

//// Support singleOutputModule

func (m *staticLibrary) outputFileName() string {
	return m.outputName() + ".a"
}

type sharedLibrary struct {
	library
	fileNameExtension string
}

// sharedLibrary supports:
// * producing output using the linker
// * producing a shared library
// * stripping symbols from output
var _ linkableModule = (*sharedLibrary)(nil)
var _ sharedLibProducer = (*sharedLibrary)(nil)
var _ stripable = (*sharedLibrary)(nil)

func (m *sharedLibrary) getLinkName() string {
	return m.outputName() + m.fileNameExtension
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

func (l *sharedLibrary) strip() bool {
	return l.Properties.Strip != nil && *l.Properties.Strip
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

//// Support singleOutputModule

func (m *sharedLibrary) outputFileName() string {
	// Since we link against libraries using the library flag style,
	// -lmod, return the name of the link library here rather than the
	// real, versioned library.
	return m.getLinkName()
}

//// Support sharedLibProducer

func (m *sharedLibrary) getTocName() string {
	return m.getRealName() + tocExt
}

type binary struct {
	library
}

// binary supports:
// * producing output using the linker
// * stripping symbols from output
var _ linkableModule = (*binary)(nil)
var _ stripable = (*binary)(nil)

func (l *binary) strip() bool {
	return l.Properties.Strip != nil && *l.Properties.Strip
}

func (m *binary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).binaryActions(m, ctx)
	}
}

//// Support singleOutputModule

func (m *binary) outputFileName() string {
	return m.outputName()
}

func (l *library) LibraryFactory(config *bobConfig, module blueprint.Module) (blueprint.Module, []interface{}) {
	l.Properties.Features.Init(&config.Properties, CommonProps{}, BuildProps{}, SplittableProps{})
	l.Properties.Host.init(&config.Properties, CommonProps{}, BuildProps{})
	l.Properties.Target.init(&config.Properties, CommonProps{}, BuildProps{})

	return module, []interface{}{&l.Properties, &l.SimpleName.Properties}
}

func staticLibraryFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &staticLibrary{}
	return module.LibraryFactory(config, module)
}

func sharedLibraryFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &sharedLibrary{}
	if config.Properties.GetBool("osx") {
		module.fileNameExtension = ".dylib"
	} else {
		module.fileNameExtension = ".so"
	}
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

func getLibrary(m blueprint.Module) (*library, bool) {
	if bsl, ok := getBinaryOrSharedLib(m); ok {
		return bsl, true
	} else if sl, ok := m.(*staticLibrary); ok {
		return &sl.library, true
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
		b.checkField(len(props.Reexport_libs) == 0, "reexport_libs")
		b.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
	} else if sl, ok := m.(*sharedLibrary); ok {
		props := sl.Properties
		sl.checkField(len(props.Export_ldflags) == 0, "export_ldflags")
	} else if sl, ok := m.(*staticLibrary); ok {
		props := sl.Properties
		sl.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
		sl.checkField(props.Version_script == nil, "version_script")
	}
}

// Check that each module only reexports libraries that it is actually using.
func checkReexportLibsMutator(mctx blueprint.TopDownMutatorContext) {
	if l, ok := getLibrary(mctx.Module()); ok {
		for _, lib := range l.Properties.Reexport_libs {
			if !utils.ListsContain(lib,
				l.Properties.Shared_libs,
				l.Properties.Static_libs,
				l.Properties.Header_libs,
				l.Properties.Whole_static_libs,
				l.Properties.Export_header_libs) {
				panic(fmt.Errorf("%s reexports unused library %s", mctx.ModuleName(), lib))
			}
		}
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
// linked explicitly (via static_libs), and another included in a different
// library via whole_static_libs.
func checkForMultipleLinking(topLevelModuleName string, staticLibs map[string]bool, insideWholeLibs map[string]string) {
	duplicateDeps := []string{}
	for dep := range staticLibs {
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
func propagateOtherExportedProperties(l *library, depLib propertyExporter) {
	props := &l.Properties.Build
	for _, shLib := range depLib.exportSharedLibs() {
		if !utils.Contains(props.Shared_libs, shLib) {
			props.Shared_libs = append(props.Shared_libs, shLib)
			props.ExtraSharedLibs = append(props.ExtraSharedLibs, shLib)
		}
	}
	for _, ldlib := range depLib.exportLdlibs() {
		if !utils.Contains(props.Ldlibs, ldlib) {
			props.Ldlibs = append(props.Ldlibs, ldlib)
		}
	}
	props.Ldflags = append(props.Ldflags, depLib.exportLdflags()...)

	// Header libraries are *not* propagated here, because they are currently
	// only supported on Android, which will automatically re-export them just
	// by adding them to LOCAL_EXPORT_HEADER_LIBRARY_HEADERS.
}

func exportLibFlagsMutator(mctx blueprint.TopDownMutatorContext) {
	l, ok := getBinaryOrSharedLib(mctx.Module())
	if !ok {
		return
	}

	// Track the set of everything mentioned in 'static_libs' of all
	// dependencies of this module, for multiple-link checking.
	allImportedStaticLibs := make(map[string]bool)
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
			for _, subLib := range depLib.Properties.Static_libs {
				allImportedStaticLibs[subLib] = true
			}

			propagateOtherExportedProperties(l, depLib)
		} else if _, ok := dep.(*generateStaticLibrary); ok {
			// Nothing to do for GeneratedStaticLibrary
			//
			// The GeneratedStaticLibrary is expected to be self
			// contained, so no pulling in of other static or shared
			// libraries.
		} else if depLib, ok := dep.(*externalLib); ok {
			propagateOtherExportedProperties(l, depLib)
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

	checkForMultipleLinking(mctx.ModuleName(), allImportedStaticLibs, insideWholeLibs)
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

	mainModuleName := mainModule.Name()

	handler.graph.AddNode(mainModuleName)

	var mainBuild *Build
	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		mainBuild = buildProps.build()
	} else {
		return // ignore not a build
	}

	for _, lib := range mainBuild.Static_libs {
		if _, err := handler.graph.AddEdgeToExistingNodes(mainModuleName, lib); err != nil {
			panic(fmt.Errorf("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModuleName, lib, lib))
		}
		handler.graph.SetEdgeColor(mainModuleName, lib, "blue")
	}

	for _, lib := range mainBuild.Whole_static_libs {
		if _, err := handler.graph.AddEdgeToExistingNodes(mainModuleName, lib); err != nil {
			panic(fmt.Errorf("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModuleName, lib, lib))
		}
		handler.graph.SetEdgeColor(mainModuleName, lib, "red")
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

	sub := graph.GetSubgraph(handler.graph, mainModuleName)

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
		cost := graph.GetSubgraphNodeCount(sub, nodeID)
		sources, _ := sub.GetSources(nodeID)
		priority := len(sources)
		sub.SetNodePriority(nodeID, (10*priority)-cost)
	}

	// The main library must always be evaluated first in the topological sort
	sub.SetNodePriority(mainModuleName, minInt)

	// We want those edges for calculating priority. After setting priority we can remove them.
	sub.DeleteProxyEdges("red")

	sub2 := graph.GetSubgraph(sub, mainModuleName)
	sortedStaticLibs, isDAG := graph.TopologicalSort(sub2)

	// Pop the module itself from the front of the list
	sortedStaticLibs = sortedStaticLibs[1:]

	if !isDAG {
		panic("We have detected cycle: " + mainModuleName)
	} else {
		mainBuild.ResolvedStaticLibs = sortedStaticLibs
	}

	extraStaticLibsDependencies := utils.Difference(mainBuild.ResolvedStaticLibs, mainBuild.Static_libs)

	mctx.AddVariationDependencies(nil, staticDepTag, extraStaticLibsDependencies...)

	// This module may now depend on extra shared libraries, inherited from included
	// static libraries. Add that dependency here.
	mctx.AddVariationDependencies(nil, sharedDepTag, mainBuild.ExtraSharedLibs...)
}
