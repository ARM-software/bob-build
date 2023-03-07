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
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/utils"
)

var depOutputsVarRegexp = regexp.MustCompile(`^\$\{(.+)_out\}$`)

// library is a base class for modules which are generated from sets of object files
type library struct {
	moduleBase
	simpleOutputProducer

	Properties struct {
		Features
		TransitiveLibraryProps
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
var _ sourceInterface = (*library)(nil)
var _ matchSourceInterface = (*library)(nil)
var _ propertyEscapeInterface = (*library)(nil)
var _ splittable = (*library)(nil)
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

func (l *library) getLegacySourceProperties() *LegacySourceProps {
	return &l.Properties.LegacySourceProps
}

func (l *library) getSourceFiles(ctx blueprint.BaseModuleContext) []string {
	return l.Properties.LegacySourceProps.getSourceFiles(ctx)
}

func (l *library) getSourceTargets(ctx blueprint.BaseModuleContext) []string {
	return l.Properties.LegacySourceProps.getSourceTargets(ctx)
}

func (l *library) getSourcesResolved(ctx blueprint.BaseModuleContext) []string {
	return l.Properties.LegacySourceProps.getSourcesResolved(ctx)
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `BuildProps` this is:
//   - Ldflags
//   - Cflags
//   - Conlyflags
//   - Cxxflags
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
				visitChildren = false
			} else if tag == staticDepTag || tag == sharedDepTag || tag == reexportLibsTag {
				/* Try to import generated header dirs from static|shared_libs too:
				 * - The library could be a bob_generate_shared_library or
				 *   bob_generate_static_library, in which case we need to import
				 *   any generated header dirs it exports.
				 * - If it's a bob_static_library or bob_shared_library, it may
				 *   export generated header dirs, so it's children need visiting.
				 */
				importHeaderDirs = true
				visitChildren = true
				// We don't know the module type so disable the check
				childMustBeGenerated = false
			}
		} else {
			if tag == exportGeneratedHeaderTag {
				importHeaderDirs = true
				visitChildren = false
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
						utils.Die("generated_headers %s must have outputs()", child.Name())
					}

					orderOnly = append(orderOnly, getHeadersGenerated(ds)...)
				}
			} else if gs, ok := getAndroidGenerateCommon(child); ok {
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
						utils.Die("generated_headers %s must have outputs()", child.Name())
					}

					orderOnly = append(orderOnly, getHeadersGenerated(ds)...)
				}
			} else if childMustBeGenerated {
				utils.Die("%s dependency on non-generated module %s", tag.(dependencyTag).name, child.Name())
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
			}
		})
	return
}

func (l *library) GetExportedVariables(ctx blueprint.ModuleContext) (expSystemIncludes, expLocalSystemIncludes, expLocalIncludes, expIncludes, expCflags []string) {
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
			expLocalSystemIncludes = append(expLocalIncludes, pe.exportLocalSystemIncludeDirs()...)
			expIncludes = append(expIncludes, pe.exportIncludeDirs()...)
			expSystemIncludes = append(expSystemIncludes, pe.exportSystemIncludeDirs()...)
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
		utils.Die("%s has field %s set", l.Name(), fieldName)
	}
}

// All libraries must implement `propertyExporter`
func (l *library) exportCflags() []string            { return l.Properties.Export_cflags }
func (l *library) exportIncludeDirs() []string       { return l.Properties.Export_include_dirs }
func (l *library) exportLocalIncludeDirs() []string  { return l.Properties.Export_local_include_dirs }
func (l *library) exportLdflags() []string           { return l.Properties.Export_ldflags }
func (l *library) exportLdlibs() []string            { return l.Properties.Ldlibs }
func (l *library) exportSharedLibs() []string        { return l.Properties.Shared_libs }
func (l *library) exportSystemIncludeDirs() []string { return l.Properties.Export_system_include_dirs }
func (l *library) exportLocalSystemIncludeDirs() []string {
	return l.Properties.Export_local_system_include_dirs
}

func (l *library) LibraryFactory(config *BobConfig, module blueprint.Module) (blueprint.Module, []interface{}) {
	l.Properties.Features.Init(&config.Properties, CommonProps{}, BuildProps{}, SplittableProps{})
	l.Properties.Host.init(&config.Properties, CommonProps{}, BuildProps{})
	l.Properties.Target.init(&config.Properties, CommonProps{}, BuildProps{})

	return module, []interface{}{&l.Properties, &l.SimpleName.Properties}
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
		b.checkField(len(props.Export_local_system_include_dirs) == 0, "export_local_system_include_dirs")
		b.checkField(len(props.Export_system_include_dirs) == 0, "export_system_include_dirs")
		b.checkField(len(props.Reexport_libs) == 0, "reexport_libs")
		b.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
	} else if sl, ok := m.(*sharedLibrary); ok {
		props := sl.Properties
		sl.checkField(len(props.Export_ldflags) == 0, "export_ldflags")
		sl.checkField(props.Mte.Memtag_heap == nil, "memtag_heap")
		sl.checkField(props.Mte.Diag_memtag_heap == nil, "memtag_heap")
	} else if sl, ok := m.(*staticLibrary); ok {
		props := sl.Properties
		sl.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
		sl.checkField(props.Version_script == nil, "version_script")
		sl.checkField(props.Mte.Memtag_heap == nil, "memtag_heap")
		sl.checkField(props.Mte.Diag_memtag_heap == nil, "memtag_heap")
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
				utils.Die("%s re-exports unused library %s", mctx.ModuleName(), lib)
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
		utils.Die(msg)
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
					utils.Die("%s links with %s and %s, which both contain %s as whole_static_libs",
						mctx.Module().Name(), firstContainingLib,
						depLib.Name(), subLib)
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
		} else if _, ok := dep.(*strictLibrary); ok {
			// TODO: Propogate flags here?
		} else {
			utils.Die("%s is not a staticLibrary", dep.Name())
		}

		// Don't add whole_static_lib components to the library list, because their
		// contents are already included in the parent library.
		if mctx.OtherModuleDependencyTag(dep) != wholeStaticDepTag && mctx.OtherModuleDependencyTag(dep) != staticDepTag {
			utils.Die("Non WholeStatic or Static dep tag encountered visiting %s from %s",
				dep.Name(), mctx.ModuleName())
		}
	})

	checkForMultipleLinking(mctx.ModuleName(), allImportedStaticLibs, insideWholeLibs)
}
