/*
 * Copyright 2018 Arm Limited.
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
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/graph"
	"github.com/ARM-software/bob-build/utils"
)

var (
	bobdir     = os.Getenv("BOB_DIR")
	srcdir     = os.Getenv("SRCDIR")
	builddir   = os.Getenv("BUILDDIR")
	configName = os.Getenv("CONFIGNAME")
	configOpts = os.Getenv("BOB_CONFIG_OPTS")
	configPath = filepath.Join(builddir, configName)
	jsonPath   = filepath.Join(builddir, "config.json")
)

// Types implementing phonyInterface support the creation of phony targets.
type phonyInterface interface {
	// A list of the outputs to be built when shortName is specified as the target
	outputs(g generatorBackend) []string

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
	sharedLibsDir(targetType string) string
	sourceOutputDir(m *generateCommon) string
	binaryOutputDir(m *binary) string
	staticLibOutputDir(m *staticLibrary) string
	sharedLibOutputDir(m *sharedLibrary) string
	kernelModOutputDir(m *kernelModule) string

	// Backend initialisation
	init(*blueprint.Context, *bobConfig)

	// Access to backend configuration
	getToolchain(tgtType string) toolchain
}

// The bobConfig type is stored against the Blueprint context, and allows us to
// retrieve the backend and configuration values from within Blueprint callbacks.
type bobConfig struct {
	Generator  generatorBackend
	Properties *configProperties
}

// getAvailableFeatures returns all available features that can be used in .bp
func (config *bobConfig) getAvailableFeatures() []string {
	return utils.SortedKeysBoolMap(config.Properties.Features)
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

func glob(ctx blueprint.ModuleContext, globs []string, excludes []string) []string {
	var files []string

	for _, file := range globs {
		if strings.ContainsAny(file, "*?[") {
			matches, _ := ctx.GlobWithDeps(file, excludes)
			files = append(files, matches...)
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
func (s *SourceProps) GetSrcs(ctx blueprint.ModuleContext) []string {
	return glob(ctx, s.Srcs, s.Exclude_srcs)
}

func (s *SourceProps) processPaths(ctx blueprint.BaseModuleContext) {
	g := getBackend(ctx)
	prefix := ctx.ModuleDir()
	var special = map[string]string{
		"${bob_config}": filepath.Join(g.buildDir(), configName),
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

// configProvider allows the retrieval of configuration
type configProvider interface {
	Config() interface{}
}

func getConfig(ctx configProvider) *bobConfig {
	return ctx.Config().(*bobConfig)
}

type dependencySingleton struct{}

func (m *dependencySingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	ctx.AddNinjaFileDeps(jsonPath)
}

func dependencySingletonFactory() blueprint.Singleton {
	return &dependencySingleton{}
}

const (
	tgtTypeHost   string = "host"
	tgtTypeTarget string = "target"
)

func stripEmptyComponentsRecursive(propsVal reflect.Value) {
	var emptyStrFilter = func(s string) bool { return s != "" }

	for i := 0; i < propsVal.NumField(); i++ {
		field := propsVal.Field(i)

		switch field.Kind() {
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				list := field.Interface().([]string)
				list = utils.Filter(list, emptyStrFilter)
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

const splitterMutatorName string = "library"

func parseAndAddVariationDeps(mctx blueprint.BottomUpMutatorContext,
	tag blueprint.DependencyTag, deps ...string) {

	hostVariation := []blueprint.Variation{blueprint.Variation{Mutator: splitterMutatorName, Variation: tgtTypeHost}}
	targetVariation := []blueprint.Variation{blueprint.Variation{Mutator: splitterMutatorName, Variation: tgtTypeTarget}}

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
var flagDepTag = dependencyTag{name: "reexport"}
var kernelModuleDepTag = dependencyTag{name: "kernel_module"}

// The targetable interface allows target-specific properties to be
// retrieved and set on a module.
type targetable interface {
	build() *Build
	features() *Features
	getTarget() string
}

func dependerMutator(mctx blueprint.BottomUpMutatorContext) {
	if e, ok := mctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	if t, ok := mctx.Module().(targetable); ok {
		build := t.build()
		if _, ok := mctx.Module().(*defaults); ok {
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
	if km, ok := mctx.Module().(*kernelModule); ok {
		mctx.AddDependency(mctx.Module(), kernelModuleDepTag, km.Properties.Extra_symbols...)
	}
	if ins, ok := mctx.Module().(installable); ok {
		props := ins.getInstallableProps()
		if props.Install_group != nil {
			mctx.AddDependency(mctx.Module(), installGroupTag, *props.Install_group)
		}
		parseAndAddVariationDeps(mctx, installDepTag, props.Install_deps...)
	}
}

// Applies target specific properties within each module. Must be done
// after the libraries have been split.
func targetMutator(mctx blueprint.TopDownMutatorContext) {
	var build *Build
	var tgtType string

	if def, ok := mctx.Module().(targetable); ok {
		build = def.build()
		tgtType = def.getTarget()
	} else if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		build = &gsc.Properties.FlagArgsBuild
		tgtType = gsc.Properties.Target
	} else {
		return
	}

	//print(mctx.ModuleName() + " is targetable\n")
	var src *TargetSpecific
	if tgtType == tgtTypeHost {
		src = &build.Host
	} else if tgtType == tgtTypeTarget {
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
	processPaths(blueprint.BaseModuleContext)
}

// Adds module paths to appropriate properties.
func pathMutator(mctx blueprint.BottomUpMutatorContext) {
	if p, ok := mctx.Module().(pathProcessor); ok {
		p.processPaths(mctx)
	}
}

func collectReexportDependenciesMutator(mctx blueprint.TopDownMutatorContext) {
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

func applyReexportDependenciesMutator(mctx blueprint.BottomUpMutatorContext) {
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
		mctx.AddVariationDependencies(nil, flagDepTag, build.ResolvedReexportedLibs...)
	}
}

// Create a closure passing the config to a module factory so that the module
// factories can access the config.
type factoryWithConfig func(*bobConfig) (blueprint.Module, []interface{})

func passConfig(mf factoryWithConfig, config *bobConfig) blueprint.ModuleFactory {
	return func() (blueprint.Module, []interface{}) {
		return mf(config)
	}
}

func findRequiredModulesMutator(ctx blueprint.TopDownMutatorContext) {
	// Non-enableable module types are aliases and defaults. All
	// dependencies of an alias should be required. Ignore defaults,
	// because they've already been applied and don't generate any build
	// rules themselves.
	if e, ok := ctx.Module().(enableable); ok {
		// If it's a top-level module (enabled and built by default), mark it as
		// required, and continue to visit its dependencies. Otherwise, we don't
		// need its dependencies so return.
		if isEnabled(e) && isBuiltByDefault(e) {
			markAsRequired(e)
		} else {
			return
		}
	} else if _, ok := ctx.Module().(*defaults); ok { // Ignore defaults.
		return
	} else if _, ok := ctx.Module().(*alias); ok { // Ignore aliases.
		return
	}

	ctx.WalkDeps(func(dep blueprint.Module, parent blueprint.Module) bool {
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

// Main is the entry point for the bob primary builder.
//
// It loads the configuration from config.json, registers the module type
// and mutators, initializes the backend, and finally calls into Blueprint.
func Main() {
	// Load the config first. This is needed because some of the module
	// types' definitions contain a struct-per-feature, and features are
	// specified in the config.
	config := &bobConfig{}
	config.Properties = loadConfig(jsonPath)

	var ctx = blueprint.NewContext()
	ctx.RegisterModuleType("bob_binary", passConfig(binaryFactory, config))
	ctx.RegisterModuleType("bob_static_library", passConfig(staticLibraryFactory, config))
	ctx.RegisterModuleType("bob_shared_library", passConfig(sharedLibraryFactory, config))
	ctx.RegisterModuleType("bob_defaults", passConfig(defaultsFactory, config))
	ctx.RegisterModuleType("bob_external_header_library", passConfig(externalLibFactory, config))
	ctx.RegisterModuleType("bob_external_shared_library", passConfig(externalLibFactory, config))
	ctx.RegisterModuleType("bob_external_static_library", passConfig(externalLibFactory, config))
	ctx.RegisterModuleType("bob_generate_source", passConfig(generateSourceFactory, config))
	ctx.RegisterModuleType("bob_transform_source", passConfig(transformSourceFactory, config))
	ctx.RegisterModuleType("bob_generate_static_library", passConfig(genStaticLibFactory, config))
	ctx.RegisterModuleType("bob_generate_shared_library", passConfig(genSharedLibFactory, config))
	ctx.RegisterModuleType("bob_generate_binary", passConfig(genBinaryFactory, config))
	ctx.RegisterModuleType("bob_alias", passConfig(aliasFactory, config))
	ctx.RegisterModuleType("bob_kernel_module", passConfig(kernelModuleFactory, config))
	ctx.RegisterModuleType("bob_resource", passConfig(resourceFactory, config))
	ctx.RegisterModuleType("bob_install_group", passConfig(installGroupFactory, config))

	// Note that the order of mutators are important, since the
	// contents of each module will be rewritten. The following
	// describes the required orderring of mutators dealing with
	// property propagation.
	//
	// Merge feature specific values to the level above in each
	// module. This must be before defaults so that a feature specific
	// option set in a default does not override an option set in a
	// module. Do this before templates so templates only need to
	// operate on one level.
	//
	// Evaluate templates next, including in defaults. This avoids us
	// having to re-evaluate templates after they have been copied
	// around by defaults.
	//
	// Then apply defaults. Do this before the library splitter so that
	// we can propagate target_supported and host_supported through
	// defaults if needed.
	//
	// Next split libraries into host and target specific modules.
	//
	// After the libraries are split we can apply target specific options.
	//
	// The depender mutator adds the dependencies between binaries and libraries.
	//
	// The generated depender mutator add dependencies to generated source modules.
	ctx.RegisterBottomUpMutator("default_deps", defaultDepsMutator).Parallel()
	ctx.RegisterTopDownMutator("features_applier", featureApplierMutator).Parallel()
	ctx.RegisterTopDownMutator("template_applier", templateApplierMutator).Parallel()
	ctx.RegisterBottomUpMutator("check_lib_fields", checkLibraryFieldsMutator).Parallel()
	ctx.RegisterBottomUpMutator("strip_empty_components", stripEmptyComponentsMutator).Parallel()
	ctx.RegisterBottomUpMutator("process_paths", pathMutator).Parallel()
	ctx.RegisterTopDownMutator("supported_variants", supportedVariantsMutator).Parallel()
	ctx.RegisterBottomUpMutator(splitterMutatorName, splitterMutator).Parallel()
	ctx.RegisterTopDownMutator("target", targetMutator).Parallel()
	ctx.RegisterTopDownMutator("default_applier", defaultApplierMutator).Parallel()
	ctx.RegisterBottomUpMutator("depender", dependerMutator).Parallel()
	ctx.RegisterBottomUpMutator("alias", aliasMutator).Parallel()
	ctx.RegisterBottomUpMutator("generated", generatedDependerMutator).Parallel()

	if handler := initGrapvizHandler(); handler != nil {
		ctx.RegisterBottomUpMutator("graphviz_output", handler.graphvizMutator)
		// Singleton for stop tool and don't overwrite build.bp
		ctx.RegisterSingletonType("quit_singleton", handler.quitSingletonFactory)
	} else {

		ctx.RegisterTopDownMutator("export_lib_flags", exportLibFlagsMutator).Parallel()
		dependencyGraphHandler := graphMutatorHandler{graph.NewGraph("All")}
		ctx.RegisterBottomUpMutator("sort_resolved_static_libs", dependencyGraphHandler.ResolveDependencySortMutator) // This can't be parallel

		ctx.RegisterTopDownMutator("find_required_modules", findRequiredModulesMutator).Parallel()

		ctx.RegisterTopDownMutator("collect_reexport_dependencies", collectReexportDependenciesMutator).Parallel()
		ctx.RegisterBottomUpMutator("apply_reexport_dependencies", applyReexportDependenciesMutator).Parallel()

		// Depend on the config file
		ctx.RegisterSingletonType("config_singleton", dependencySingletonFactory)
	}

	if config.Properties.GetBool("builder_linux") {
		config.Generator = &linuxGenerator{}
	} else if config.Properties.GetBool("builder_android") {
		config.Generator = &androidMkGenerator{}
	} else {
		panic(errors.New("unknown builder backend"))
	}

	config.Generator.init(ctx, config)
	bootstrap.Main(ctx, config)
}
