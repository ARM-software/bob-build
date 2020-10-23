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
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/utils"
)

// Types implementing phonyInterface support the creation of phony targets.
type phonyInterface interface {
	// A list of the outputs to be built when shortName is specified as the target
	outputs() []string
	implicitOutputs() []string

	// The name of the target that can be used
	shortName() string
}

// Types implementing moduleWithBuildProps support all the compiler build
// properties.
type moduleWithBuildProps interface {
	build() *Build
}

// A TargetSpecific module is one that supports building on host and target,
// with a set of properties in `host: {}` or `target: {}` blocks.
type TargetSpecific struct {
	Features

	// 'BlueprintEmbed' is a special case in Blueprint which makes it interpret
	// a runtime-generated type as being embedded in its parent struct.
	BlueprintEmbed interface{}
}

// init initializes properties and features
func (t *TargetSpecific) init(properties *configProperties, list ...interface{}) {
	if len(list) == 0 {
		panic("List can't be empty")
	}

	propsType := coalesceTypes(typesOf(list...)...)
	t.BlueprintEmbed = reflect.New(propsType).Interface()

	t.Features.Init(properties, list...)
}

// getTargetSpecificProps returns target specific property data as an empty interface
func (t *TargetSpecific) getTargetSpecificProps() interface{} {
	return t.BlueprintEmbed
}

// A type implementing dependentInterface can be depended upon by other modules.
type dependentInterface interface {
	phonyInterface
	outputDir() string
}

// dependencyTag contains the name of the tag used to track a particular type
// of dependency between modules
type dependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}

func getBackend(ctx blueprint.BaseModuleContext) generatorBackend {
	return getConfig(ctx).Generator
}

// A generatorBackend outputs build definitions for a given backend for each
// supported module type. There are also support functions to identify
// backend specific information
type generatorBackend interface {
	// Module build actions
	aliasActions(*alias, blueprint.ModuleContext)
	binaryActions(*binary, blueprint.ModuleContext)
	generateSourceActions(*generateSource, blueprint.ModuleContext)
	transformSourceActions(*transformSource, blueprint.ModuleContext)
	genSharedActions(*generateSharedLibrary, blueprint.ModuleContext)
	genStaticActions(*generateStaticLibrary, blueprint.ModuleContext)
	genBinaryActions(*generateBinary, blueprint.ModuleContext)
	kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext)
	sharedActions(*sharedLibrary, blueprint.ModuleContext)
	staticActions(*staticLibrary, blueprint.ModuleContext)
	resourceActions(*resource, blueprint.ModuleContext)

	// Backend specific info for module types
	buildDir() string
	sourceDir() string
	bobScriptsDir() string
	sharedLibsDir(tgt tgtType) string

	// Backend flag escaping
	escapeFlag(string) string

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

// AndroidProps defines module properties used by Android backends
type AndroidProps struct {
	// Values to use on Android for LOCAL_MODULE_TAGS, defining which builds this module is built for
	Tags []string
	// Value to use on Android for LOCAL_MODULE_OWNER
	Owner string
}

func (p *AndroidProps) isProprietary() bool {
	return p.Owner != ""
}

// AndroidPGOProps defines properties used to support profile-guided optimization.
type AndroidPGOProps struct {
	Pgo struct {
		Benchmarks         []string
		Profile_file       *string
		Enable_profile_use *bool
		Cflags             []string
	}
}

func getBobScriptsDir() string {
	return filepath.Join(getBobDir(), "scripts")
}

// Construct a path to a file within the build directory that Go can
// use to create a file.
//
// This is _not_ intended for use in writing ninja rules.
func getPathInBuildDir(elems ...string) string {
	return filepath.Join(append([]string{getBuildDir()}, elems...)...)
}

// Construct a path to a file within the source directory that Go can
// use to create a file.
//
// This is _not_ intended for use in writing ninja rules.
func getPathInSourceDir(elems ...string) string {
	return filepath.Join(append([]string{getSourceDir()}, elems...)...)
}

// Construct paths to files within the source directory that Go can
// use to create files.
//
// This is _not_ intended for use in writing ninja rules.
func getPathsInSourceDir(filelist []string) []string {
	return utils.PrefixDirs(filelist, getSourceDir())
}

// Construct a path to a file within the build directory to be used
// in backend output files.
func getBackendPathInBuildDir(g generatorBackend, elems ...string) string {
	return filepath.Join(append([]string{g.buildDir()}, elems...)...)
}

// Construct a path to a file within the source directory to be used
// in backend output files.
func getBackendPathInSourceDir(g generatorBackend, elems ...string) string {
	return filepath.Join(append([]string{g.sourceDir()}, elems...)...)
}

// Construct paths to files within the source directory to be used in
// backend output files.
func getBackendPathsInSourceDir(g generatorBackend, filelist []string) []string {
	return utils.PrefixDirs(filelist, g.sourceDir())
}

// Construct a path to a file within the scripts directory to be used
// in backend output files.
func getBackendPathInBobScriptsDir(g generatorBackend, elems ...string) string {
	return filepath.Join(append([]string{g.bobScriptsDir()}, elems...)...)
}

func glob(ctx blueprint.BaseModuleContext, globs []string, excludes []string) []string {
	var files []string

	// If any globs are used, we need to use an exclude list which is
	// relative to the source directory.
	excludesFromSrcDir := getPathsInSourceDir(excludes)

	for _, file := range globs {
		if strings.ContainsAny(file, "*?[") {
			// Globs need to be calculated relative to the source
			// directory (not the working directory), so add it
			// here, and remove it afterwards.
			file = getPathInSourceDir(file)
			matches, _ := ctx.GlobWithDeps(file, excludesFromSrcDir)
			for _, match := range matches {
				rel, err := filepath.Rel(getSourceDir(), match)
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

// Get a list of sources to compile.
//
// The sources are relative to the project directory (i.e. include
// the module directory but not the base source directory), and
// excludes have been handled.
func (s *SourceProps) getSources(ctx blueprint.BaseModuleContext) []string {
	return glob(ctx, s.Srcs, s.Exclude_srcs)
}

func (s *SourceProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	prefix := projectModuleDir(ctx)
	var special = map[string]string{
		"${bob_config}":      configFile,
		"${bob_config_json}": filepath.Join(getBuildDir(), configJSONFile),
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

func stripEmptyComponents(list []string) []string {
	var emptyStrFilter = func(s string) bool { return s != "" }

	return utils.Filter(emptyStrFilter, list)
}

func stripEmptyComponentsRecursive(propsVal reflect.Value) {

	for i := 0; i < propsVal.NumField(); i++ {
		field := propsVal.Field(i)

		switch field.Kind() {
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				list := field.Interface().([]string)
				list = stripEmptyComponents(list)
				field.Set(reflect.ValueOf(list))
			}

		case reflect.Struct:
			stripEmptyComponentsRecursive(field)
		}
	}
}

func stripEmptyComponentsMutator(mctx blueprint.BottomUpMutatorContext) {
	f, ok := mctx.Module().(featurable)
	if !ok {
		return
	}

	strippableProps := f.featurableProperties()

	if t, ok := mctx.Module().(targetSpecificLibrary); ok {
		for _, tgt := range []tgtType{tgtTypeHost, tgtTypeTarget} {
			tgtSpecific := t.getTargetSpecific(tgt)
			tgtSpecificData := tgtSpecific.getTargetSpecificProps()
			strippableProps = append(strippableProps, tgtSpecificData)
		}
	}

	for _, props := range strippableProps {
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

func parseAndAddVariationDeps(mctx blueprint.BottomUpMutatorContext,
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
			mctx.AddDependency(mctx.Module(), tag, dep)
		}
	}
}

var wholeStaticDepTag = dependencyTag{name: "whole_static"}
var headerDepTag = dependencyTag{name: "header"}
var staticDepTag = dependencyTag{name: "static"}
var sharedDepTag = dependencyTag{name: "shared"}
var reexportLibsTag = dependencyTag{name: "reexport_libs"}
var kernelModuleDepTag = dependencyTag{name: "kernel_module"}

func dependerMutator(mctx blueprint.BottomUpMutatorContext) {
	if e, ok := mctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	if l, ok := getLibrary(mctx.Module()); ok {
		build := &l.Properties.Build

		mctx.AddVariationDependencies(nil, wholeStaticDepTag, build.Whole_static_libs...)
		mctx.AddVariationDependencies(nil, staticDepTag, build.Static_libs...)

		mctx.AddVariationDependencies(nil, headerDepTag, build.Header_libs...)
		mctx.AddVariationDependencies(nil, headerDepTag, build.Export_header_libs...)

		mctx.AddVariationDependencies(nil, sharedDepTag, build.Shared_libs...)
	}

	if km, ok := mctx.Module().(*kernelModule); ok {
		mctx.AddDependency(mctx.Module(), kernelModuleDepTag, km.Properties.Extra_symbols...)
	}

	if ins, ok := mctx.Module().(installable); ok {
		props := ins.getInstallableProps()
		if props.Install_group != nil {
			mctx.AddDependency(mctx.Module(), installGroupTag, proptools.String(props.Install_group))
		}
		parseAndAddVariationDeps(mctx, installDepTag, props.Install_deps...)
	}
	if strlib, ok := mctx.Module().(stripable); ok {
		info := strlib.getDebugInfo()
		if info != nil {
			mctx.AddDependency(mctx.Module(), debugInfoTag, *info)
		}
	}
}

// Applies target specific properties within each module. Must be done
// after the libraries have been split.
func targetMutator(mctx blueprint.TopDownMutatorContext) {
	var build *Build
	var tgt tgtType

	if l, ok := getLibrary(mctx.Module()); ok {
		build = &l.Properties.Build
		tgt = l.Properties.TargetType
	} else if d, ok := mctx.Module().(*defaults); ok {
		build = &d.Properties.Build
		tgt = d.Properties.TargetType
	} else if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		build = &gsc.Properties.FlagArgsBuild
		tgt = gsc.Properties.Target
	} else {
		return
	}

	if tgt != tgtTypeHost && tgt != tgtTypeTarget {
		// This is fine - it can happen if the target is the default
		return
	}

	targetProps := build.getTargetSpecific(tgt).getTargetSpecificProps()

	// Copy the target-specific variables to the core set
	err := proptools.AppendMatchingProperties([]interface{}{&build.BuildProps}, targetProps, nil)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			panic(err)
		}
	}
}

type pathProcessor interface {
	processPaths(blueprint.BaseModuleContext, generatorBackend)
}

// Adds module paths to appropriate properties.
func pathMutator(mctx blueprint.BottomUpMutatorContext) {
	if p, ok := mctx.Module().(pathProcessor); ok {
		p.processPaths(mctx, getBackend(mctx))
	}
}

func collectReexportLibsDependenciesMutator(mctx blueprint.TopDownMutatorContext) {
	mainModule := mctx.Module()
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

	mctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		depTag := mctx.OtherModuleDependencyTag(child)
		if depTag == wholeStaticDepTag || depTag == staticDepTag || depTag == sharedDepTag {
			parentModule, ok1 := parent.(moduleWithBuildProps)
			childModule, ok2 := child.(moduleWithBuildProps)

			if !ok1 || !ok2 {
				return false
			}

			parentBuild := parentModule.build()
			childBuild := childModule.build()

			if len(childBuild.Reexport_libs) > 0 &&
				(parent.Name() == mainModule.Name() || utils.Contains(parentBuild.Reexport_libs, child.Name())) {
				mainBuild.ResolvedReexportedLibs = utils.AppendUnique(mainBuild.ResolvedReexportedLibs, childBuild.Reexport_libs)
				return true
			}
		}

		return false
	})
}

func applyReexportLibsDependenciesMutator(mctx blueprint.BottomUpMutatorContext) {
	mainModule := mctx.Module()
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

func findRequiredModulesMutator(mctx blueprint.TopDownMutatorContext) {
	// Non-enableable module types are aliases and defaults. All
	// dependencies of an alias should be required. Ignore defaults,
	// because they've already been applied and don't generate any build
	// rules themselves.
	if e, ok := mctx.Module().(enableable); ok {
		// If it's a top-level module (enabled and built by default), mark it as
		// required, and continue to visit its dependencies. Otherwise, we don't
		// need its dependencies so return.
		if isEnabled(e) && isBuiltByDefault(e) {
			markAsRequired(e)
		} else {
			return
		}
	} else if _, ok := mctx.Module().(*defaults); ok { // Ignore defaults.
		return
	} else if _, ok := mctx.Module().(*alias); ok { // Ignore aliases.
		return
	}

	mctx.WalkDeps(func(dep blueprint.Module, parent blueprint.Module) bool {
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

func checkDisabledMutator(mctx blueprint.BottomUpMutatorContext) {
	module := mctx.Module()
	// Skip if already disabled, or if defaults type,
	// or if type is not enableable (eg. alias)
	ep, ok := module.(enableable)
	if ok {
		if _, ok := module.(*defaults); ok {
			return
		}
		if !isEnabled(ep) {
			return
		}
	} else {
		return
	}

	// check if any direct dependency is disabled
	disabledDeps := []string{}

	mctx.VisitDirectDeps(func(dep blueprint.Module) {
		// ignore defaults - it's allowed for them to be disabled
		if _, ok := dep.(*defaults); ok {
			return
		}
		if e, ok := dep.(enableable); ok {
			if !isEnabled(e) {
				disabledDeps = utils.AppendIfUnique(disabledDeps, dep.Name())
			}
		}
	})

	// disable current module if dependency is disabled, or panic if it's required
	if len(disabledDeps) > 0 {
		if isRequired(ep) {
			panic(fmt.Errorf("Module %s is required but depends on disabled modules %s", module.Name(), strings.Join(disabledDeps, ", ")))
		} else {
			ep.getEnableableProps().Enabled = proptools.BoolPtr(false)
			return
		}
	}
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
