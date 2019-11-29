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
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/abstr"
	"github.com/ARM-software/bob-build/utils"
)

// Types implementing phonyInterface support the creation of phony targets.
type phonyInterface interface {
	// A list of the outputs to be built when shortName is specified as the target
	outputs(g generatorBackend) []string
	implicitOutputs(g generatorBackend) []string

	// The name of the target that can be used
	shortName() string
}

// Types implementing moduleWithBuildProps support all the compiler build
// properties.
type moduleWithBuildProps interface {
	build() *Build
}

// A TargetSpecific module is one that supports building on host and target.
type TargetSpecific struct {
	Features
	BuildProps
}

// A type implementing dependentInterface can be depended upon by other modules.
type dependentInterface interface {
	phonyInterface
	outputDir(g generatorBackend) string
}

// dependencyTag contains the name of the tag used to track a particular type
// of dependency between modules
type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

func getBackend(ctx abstr.BaseModuleContext) generatorBackend {
	return getConfig(ctx).Generator
}

// A generatorBackend outputs build definitions for a given backend for each
// supported module type. There are also support functions to identify
// backend specific information
type generatorBackend interface {
	// Module build actions
	aliasActions(*alias, blueprint.ModuleContext)
	binaryActions(*binary, blueprint.ModuleContext)
	generateSourceActions(*generateSource, blueprint.ModuleContext, []inout)
	transformSourceActions(*transformSource, blueprint.ModuleContext, []inout)
	genSharedActions(*generateSharedLibrary, blueprint.ModuleContext, []inout)
	genStaticActions(*generateStaticLibrary, blueprint.ModuleContext, []inout)
	genBinaryActions(*generateBinary, blueprint.ModuleContext, []inout)
	kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext)
	sharedActions(*sharedLibrary, blueprint.ModuleContext)
	staticActions(*staticLibrary, blueprint.ModuleContext)
	resourceActions(*resource, blueprint.ModuleContext)

	// Backend specific info for module types
	buildDir() string
	sourcePrefix() string
	sharedLibsDir(tgt tgtType) string
	sourceOutputDir(m *generateCommon) string
	binaryOutputDir(m *binary) string
	staticLibOutputDir(m *staticLibrary) string
	sharedLibOutputDir(m *sharedLibrary) string
	kernelModOutputDir(m *kernelModule) string

	// Backend initialisation
	init(*blueprint.Context, *bobConfig)

	// Access to backend configuration
	getToolchain(tgt tgtType) toolchain
}

// The bobConfig type is stored against the Blueprint context, and allows us to
// retrieve the backend and configuration values from within Blueprint callbacks.
type bobConfig struct {
	Generator  generatorBackend
	Properties configProperties
}

// SourceProps defines module properties that are used to identify the
// source files associated with a module.
type SourceProps struct {
	// The list of source files. Wildcards can be used (but are suboptimal)
	Srcs []string
	// The list of source files that should not be included. Use with care.
	Exclude_srcs []string

	// Sources that we need to treat specially
	Specials []string `blueprint:"mutated"`
}

func glob(ctx abstr.BaseModuleContext, globs []string, excludes []string) []string {
	var files []string

	// If any globs are used, we need to use an exclude list which is
	// relative to the source directory.
	excludesFromSrcDir := utils.PrefixDirs(excludes, srcdir)

	for _, file := range globs {
		if strings.ContainsAny(file, "*?[") {
			// Globs need to be calculated relative to the source
			// directory (not the working directory), so add it
			// here, and remove it afterwards.
			file = filepath.Join(srcdir, file)
			matches, _ := ctx.GlobWithDeps(file, excludesFromSrcDir)
			for _, match := range matches {
				rel, err := filepath.Rel(srcdir, match)
				if err != nil {
					panic(err)
				}
				files = append(files, rel)
			}
		} else if !utils.Contains(excludes, file) {
			files = append(files, file)
		}
	}
	return files
}

// Get a list of sources to compile.
//
// The sources are relative to the project directory (i.e. include
// the module directory but not the base source directory), and
// excludes have been handled.
func (s *SourceProps) getSources(ctx abstr.BaseModuleContext) []string {
	return glob(ctx, s.Srcs, s.Exclude_srcs)
}

func (s *SourceProps) processPaths(ctx abstr.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)
	var special = map[string]string{
		"${bob_config}": configFile,
	}

	// Look for special items. Remove from Srcs and add to Specials
	srcs := []string{}
	for _, src := range s.Srcs {
		if value, ok := special[src]; !ok {
			srcs = append(srcs, src)
		} else {
			// Only append if not in Excluded.
			// Users shouldn't rely on how any special is expanded, so
			// no need to check the expanded case.
			if !utils.Contains(s.Exclude_srcs, src) {
				s.Specials = append(s.Specials, value)
			}
		}
	}

	s.Srcs = utils.PrefixDirs(srcs, prefix)
	s.Exclude_srcs = utils.PrefixDirs(s.Exclude_srcs, prefix)
}

type tgtType string

const (
	tgtTypeHost    tgtType = "host"
	tgtTypeTarget  tgtType = "target"
	tgtTypeUnknown tgtType = ""
)

func stripEmptyComponentsRecursive(propsVal reflect.Value) {
	var emptyStrFilter = func(s string) bool { return s != "" }

	for i := 0; i < propsVal.NumField(); i++ {
		field := propsVal.Field(i)

		switch field.Kind() {
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				list := field.Interface().([]string)
				list = utils.Filter(emptyStrFilter, list)
				field.Set(reflect.ValueOf(list))
			}

		case reflect.Struct:
			stripEmptyComponentsRecursive(field)
		}
	}
}

func stripEmptyComponentsMutator(mctx abstr.BottomUpMutatorContext) {
	f, ok := abstr.Module(mctx).(featurable)
	if !ok {
		return
	}

	for _, props := range f.topLevelProperties() {
		propsVal := reflect.Indirect(reflect.ValueOf(props))
		stripEmptyComponentsRecursive(propsVal)
	}
}

func moduleNamesFromLibList(libList []string) (ret []string) {
	moduleNames := make(map[string]bool)

	for _, lib := range libList {
		module, _ := splitGeneratedComponent(lib)
		if _, ok := moduleNames[module]; !ok {
			ret = append(ret, module)
			moduleNames[module] = true
		}
	}

	return ret
}

const splitterMutatorName string = "bob_splitter"

func parseAndAddVariationDeps(mctx abstr.BottomUpMutatorContext,
	tag blueprint.DependencyTag, deps ...string) {

	hostVariation := []blueprint.Variation{blueprint.Variation{Mutator: splitterMutatorName, Variation: string(tgtTypeHost)}}
	targetVariation := []blueprint.Variation{blueprint.Variation{Mutator: splitterMutatorName, Variation: string(tgtTypeTarget)}}

	for _, dep := range deps {
		var variations []blueprint.Variation

		idx := strings.LastIndex(dep, ":")
		if idx > 0 {
			variationNames := strings.Split(dep[idx+1:len(dep)], ",")
			for _, vn := range variationNames {
				if vn == "host" {
					variations = append(variations, hostVariation...)
				} else if vn == "target" {
					variations = append(variations, targetVariation...)
				} else {
					panic("Invalid variation: " + vn + " in module name " + dep)
				}
			}

			dep = dep[0:idx]
		}

		if len(variations) > 0 {
			mctx.AddVariationDependencies(variations, tag, dep)
		} else {
			mctx.AddDependency(abstr.Module(mctx), tag, dep)
		}
	}
}

var wholeStaticDepTag = dependencyTag{name: "whole_static"}
var headerDepTag = dependencyTag{name: "header"}
var staticDepTag = dependencyTag{name: "static"}
var sharedDepTag = dependencyTag{name: "shared"}
var reexportLibsTag = dependencyTag{name: "reexport_libs"}
var kernelModuleDepTag = dependencyTag{name: "kernel_module"}

// The targetable interface allows target-specific properties to be
// retrieved and set on a module.
type targetable interface {
	build() *Build
	features() *Features
	getTarget() tgtType
}

func dependerMutator(mctx abstr.BottomUpMutatorContext) {
	if e, ok := abstr.Module(mctx).(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	if t, ok := abstr.Module(mctx).(targetable); ok {
		build := t.build()
		if _, ok := abstr.Module(mctx).(*defaults); ok {
			// We do not want to add dependencies for defaults
			return
		}
		mctx.AddVariationDependencies(nil, wholeStaticDepTag, build.Whole_static_libs...)
		mctx.AddVariationDependencies(nil, staticDepTag, build.Static_libs...)
		mctx.AddVariationDependencies(nil, staticDepTag, build.Export_static_libs...)

		mctx.AddVariationDependencies(nil, headerDepTag, build.Header_libs...)
		mctx.AddVariationDependencies(nil, headerDepTag, build.Export_header_libs...)

		mctx.AddVariationDependencies(nil, sharedDepTag, build.Shared_libs...)
		mctx.AddVariationDependencies(nil, sharedDepTag, build.Export_shared_libs...)
	}
	if km, ok := abstr.Module(mctx).(*kernelModule); ok {
		mctx.AddDependency(abstr.Module(mctx), kernelModuleDepTag, km.Properties.Extra_symbols...)
	}
	if ins, ok := abstr.Module(mctx).(installable); ok {
		props := ins.getInstallableProps()
		if props.Install_group != nil {
			mctx.AddDependency(abstr.Module(mctx), installGroupTag, proptools.String(props.Install_group))
		}
		parseAndAddVariationDeps(mctx, installDepTag, props.Install_deps...)
	}
	if strlib, ok := abstr.Module(mctx).(stripable); ok {
		info := strlib.getDebugInfo()
		if info != nil {
			mctx.AddDependency(abstr.Module(mctx), debugInfoTag, *info)
		}
	}
}

// Applies target specific properties within each module. Must be done
// after the libraries have been split.
func targetMutator(mctx abstr.TopDownMutatorContext) {
	var build *Build
	var tgt tgtType

	if def, ok := abstr.Module(mctx).(targetable); ok {
		build = def.build()
		tgt = def.getTarget()
	} else if gsc, ok := getGenerateCommon(abstr.Module(mctx)); ok {
		build = &gsc.Properties.FlagArgsBuild
		tgt = gsc.Properties.Target
	} else {
		return
	}

	//print(mctx.ModuleName() + " is targetable\n")
	var src *TargetSpecific
	if tgt == tgtTypeHost {
		src = &build.Host
	} else if tgt == tgtTypeTarget {
		src = &build.Target
	} else {
		// This is fine - it can happen if the target is the default
		return
	}

	// Copy the target-specific variables to the core set
	err := proptools.AppendMatchingProperties([]interface{}{&build.BuildProps}, &src.BuildProps, nil)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			panic(err)
		}
	}
}

type pathProcessor interface {
	processPaths(abstr.BaseModuleContext, generatorBackend)
}

// Adds module paths to appropriate properties.
func pathMutator(mctx abstr.BottomUpMutatorContext) {
	if p, ok := abstr.Module(mctx).(pathProcessor); ok {
		p.processPaths(mctx, getBackend(mctx))
	}
}

type buildWrapperProcessor interface {
	processBuildWrapper(blueprint.BaseModuleContext)
}

// Prefixes build_wrapper with source path if necessary
func buildWrapperMutator(mctx blueprint.BottomUpMutatorContext) {
	if p, ok := abstr.Module(mctx).(buildWrapperProcessor); ok {
		p.processBuildWrapper(mctx)
	}
}

func collectReexportLibsDependenciesMutator(mctx abstr.TopDownMutatorContext) {
	mainModule := abstr.Module(mctx)
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			return // Not enabled, so don't add dependencies
		}
	}

	var mainBuild *Build
	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		mainBuild = buildProps.build()
	} else {
		return // We do not want to add dependencies for not targets
	}

	abstr.WalkDeps(mctx, func(child blueprint.Module, parent blueprint.Module) bool {
		depTag := mctx.OtherModuleDependencyTag(child)
		if depTag == wholeStaticDepTag || depTag == staticDepTag || depTag == sharedDepTag {
			var parentBuild *Build
			if moduleWithBuildProps, ok := parent.(moduleWithBuildProps); ok {
				parentBuild = moduleWithBuildProps.build()
			} else {
				return false
			}

			var childBuild *Build
			if moduleWithBuildProps, ok := child.(moduleWithBuildProps); ok {
				childBuild = moduleWithBuildProps.build()
			} else {
				return false
			}

			if len(childBuild.Reexport_libs) > 0 &&
				(parent.Name() == mainModule.Name() || utils.Contains(parentBuild.Reexport_libs, child.Name())) {
				mainBuild.ResolvedReexportedLibs = utils.AppendUnique(mainBuild.ResolvedReexportedLibs, childBuild.Reexport_libs)
				return true
			}
		}

		return false
	})
}

func applyReexportLibsDependenciesMutator(mctx abstr.BottomUpMutatorContext) {
	mainModule := abstr.Module(mctx)
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	var build *Build
	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		build = buildProps.build()
		mctx.AddVariationDependencies(nil, reexportLibsTag, build.ResolvedReexportedLibs...)
	}
}

func findRequiredModulesMutator(mctx abstr.TopDownMutatorContext) {
	// Non-enableable module types are aliases and defaults. All
	// dependencies of an alias should be required. Ignore defaults,
	// because they've already been applied and don't generate any build
	// rules themselves.
	if e, ok := abstr.Module(mctx).(enableable); ok {
		// If it's a top-level module (enabled and built by default), mark it as
		// required, and continue to visit its dependencies. Otherwise, we don't
		// need its dependencies so return.
		if isEnabled(e) && isBuiltByDefault(e) {
			markAsRequired(e)
		} else {
			return
		}
	} else if _, ok := abstr.Module(mctx).(*defaults); ok { // Ignore defaults.
		return
	} else if _, ok := abstr.Module(mctx).(*alias); ok { // Ignore aliases.
		return
	}

	abstr.WalkDeps(mctx, func(dep blueprint.Module, parent blueprint.Module) bool {
		e, ok := dep.(enableable)
		if ok {
			// Stop traversing if we've already visited this while
			// following another module's dependencies.
			if isRequired(e) {
				return false
			}
			// Don't require disabled dependencies (for example aliases with
			// some disabled sources).
			if !isEnabled(e) {
				return false
			}
			markAsRequired(e)
		}
		return true
	})
}

type factoryWithConfig func(*bobConfig) (blueprint.Module, []interface{})

func registerModuleTypes(register func(string, factoryWithConfig)) {
	register("bob_binary", binaryFactory)
	register("bob_static_library", staticLibraryFactory)
	register("bob_shared_library", sharedLibraryFactory)

	register("bob_defaults", defaultsFactory)

	register("bob_external_header_library", externalLibFactory)
	register("bob_external_shared_library", externalLibFactory)
	register("bob_external_static_library", externalLibFactory)

	register("bob_generate_source", generateSourceFactory)
	register("bob_transform_source", transformSourceFactory)
	register("bob_generate_static_library", genStaticLibFactory)
	register("bob_generate_shared_library", genSharedLibFactory)
	register("bob_generate_binary", genBinaryFactory)

	register("bob_alias", aliasFactory)
	register("bob_kernel_module", kernelModuleFactory)
	register("bob_resource", resourceFactory)
	register("bob_install_group", installGroupFactory)
}
