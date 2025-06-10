package core

import (
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
)

// Types implementing phonyInterface support the creation of phony targets.
type phonyInterface interface {
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
func (t *TargetSpecific) init(properties *config.Properties, list ...interface{}) {
	if len(list) == 0 {
		utils.Die("List can't be empty")
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
// TODO: Delete this interface and move over all usage to File providers.
// All instances of `outputs` should eventually be replaced. For now they
// are implemented by the file provider under the hood.
type dependentInterface interface {
	phonyInterface
	file.Provider
}

func getGenerator(ctx blueprint.BaseModuleContext) generatorBackend {
	return getConfig(ctx).Generator
}

// A generatorBackend outputs build definitions for a given backend for each
// supported module type. There are also support functions to identify
// backend specific information
type generatorBackend interface {
	// Module build actions
	aliasActions(*ModuleAlias, blueprint.ModuleContext)
	binaryActions(*ModuleBinary, blueprint.ModuleContext)
	generateSourceActions(*ModuleGenerateSource, blueprint.ModuleContext)
	genruleActions(*ModuleGenrule, blueprint.ModuleContext)
	gensrcsActions(*ModuleGensrcs, blueprint.ModuleContext)
	transformSourceActions(*ModuleTransformSource, blueprint.ModuleContext)
	genSharedActions(*generateSharedLibrary, blueprint.ModuleContext)
	genStaticActions(*generateStaticLibrary, blueprint.ModuleContext)
	genBinaryActions(*generateBinary, blueprint.ModuleContext)
	kernelModuleActions(*ModuleKernelObject, blueprint.ModuleContext)
	sharedActions(*ModuleSharedLibrary, blueprint.ModuleContext)
	staticActions(*ModuleStaticLibrary, blueprint.ModuleContext)
	resourceActions(*ModuleResource, blueprint.ModuleContext)
	filegroupActions(*ModuleFilegroup, blueprint.ModuleContext)
	strictBinaryActions(*ModuleStrictBinary, blueprint.ModuleContext)
	strictLibraryActions(*ModuleStrictLibrary, blueprint.ModuleContext)
	executableTestActions(*ModuleTest, blueprint.ModuleContext)
}

// The `BobConfig` type is stored against the Blueprint context, and allows us to
// retrieve the backend and configuration values from within Blueprint callbacks.
type BobConfig struct {
	Generator  generatorBackend
	Properties config.Properties
}

// AndroidProps defines module properties used by Android backends
type AndroidProps struct {
	// Values to use on Android for LOCAL_MODULE_TAGS, defining which builds this module is built for
	Tags []string
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

// AndroidMTEProps defines properties used to enable the Arm Memory Tagging Extension
type AndroidMTEProps struct {
	Mte struct {
		Memtag_heap      *bool
		Diag_memtag_heap *bool
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
	return filepath.Join(append([]string{backend.Get().BuildDir()}, elems...)...)
}

// Construct a path to a file within the source directory to be used
// in backend output files.
func getBackendPathInSourceDir(g generatorBackend, elems ...string) string {
	return filepath.Join(append([]string{backend.Get().SourceDir()}, elems...)...)
}

// Construct paths to files within the source directory to be used in
// backend output files.
func getBackendPathsInSourceDir(g generatorBackend, filelist []string) []string {
	return utils.PrefixDirs(filelist, backend.Get().SourceDir())
}

// Construct a path to a file within the scripts directory to be used
// in backend output files.
func getBackendPathInBobScriptsDir(g generatorBackend, elems ...string) string {
	return filepath.Join(append([]string{backend.Get().BobScriptsDir()}, elems...)...)
}

// TODO: Add support for directories.
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
			matches, err := ctx.GlobWithDeps(file, excludesFromSrcDir)

			if err != nil {
				ctx.ModuleErrorf("glob failed with: %s", err)
			}

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

// IncludeDirsProps defines a set of properties for including directories
// by the module.
type IncludeDirsProps struct {
	// The list of include dirs to use that is relative to the source directory
	Include_dirs []string `bob:"first_overrides"`

	// The list of include dirs to use that is relative to the build.bp file
	// These use relative instead of absolute paths
	Local_include_dirs []string `bob:"first_overrides"`
}

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

func stripEmptyComponentsMutator(ctx blueprint.BottomUpMutatorContext) {
	f, ok := ctx.Module().(Featurable)
	if !ok {
		return
	}

	strippableProps := f.FeaturableProperties()

	if t, ok := ctx.Module().(targetSpecificLibrary); ok {
		for _, tgt := range []toolchain.TgtType{toolchain.TgtTypeHost, toolchain.TgtTypeTarget} {
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

func parseAndAddVariationDeps(ctx blueprint.BottomUpMutatorContext,
	tag blueprint.DependencyTag, deps ...string) {

	hostVariation := []blueprint.Variation{{Mutator: splitterMutatorName, Variation: string(toolchain.TgtTypeHost)}}
	targetVariation := []blueprint.Variation{{Mutator: splitterMutatorName, Variation: string(toolchain.TgtTypeTarget)}}

	for _, dep := range deps {
		var variations []blueprint.Variation

		idx := strings.LastIndex(dep, ":")
		if idx > 0 {
			variationNames := strings.Split(dep[idx+1:], ",")
			for _, vn := range variationNames {
				if vn == "host" {
					variations = append(variations, hostVariation...)
				} else if vn == "target" {
					variations = append(variations, targetVariation...)
				} else {
					utils.Die("Invalid variation: %s in module name %s", vn, dep)
				}
			}

			dep = dep[0:idx]
		}

		if len(variations) > 0 {
			ctx.AddVariationDependencies(variations, tag, dep)
		} else {
			ctx.AddDependency(ctx.Module(), tag, dep)
		}
	}
}

func dependerMutator(ctx blueprint.BottomUpMutatorContext) {
	if e, ok := ctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	if m, ok := ctx.Module().(file.Provider); ok {
		ctx.AddDependency(ctx.Module(), tag.FilegroupTag, m.OutFileTargets()...)
	}

	if m, ok := ctx.Module().(file.Consumer); ok {
		ctx.AddDependency(ctx.Module(), tag.FilegroupTag, m.GetTargets()...)
	}

	if l, ok := getLibrary(ctx.Module()); ok {
		build := &l.Properties.Build

		ctx.AddVariationDependencies(nil, tag.WholeStaticTag, build.Whole_static_libs...)
		ctx.AddVariationDependencies(nil, tag.StaticTag, build.Static_libs...)

		ctx.AddVariationDependencies(nil, tag.HeaderTag, build.Header_libs...)
		ctx.AddVariationDependencies(nil, tag.HeaderTag, build.Export_header_libs...)

		ctx.AddVariationDependencies(nil, tag.SharedTag, build.Shared_libs...)
	}

	toolchainDefault := ToolchainModuleMap.Get(ctx.ModuleDir())

	// TODO: refactor this given the duplicated code pattern here.
	if sl, ok := ctx.Module().(*ModuleStrictLibrary); ok {
		ctx.AddVariationDependencies(nil, tag.DepTag, sl.Properties.Deps...)
		toolchain := proptools.StringDefault(sl.Properties.Toolchain, toolchainDefault)
		if toolchain != "" {
			ctx.AddVariationDependencies(nil, tag.ToolchainTag, toolchain)
		}
	}

	if sb, ok := ctx.Module().(*ModuleStrictBinary); ok {
		ctx.AddVariationDependencies(nil, tag.DepTag, sb.Properties.Deps...)
		toolchain := proptools.StringDefault(sb.Properties.Toolchain, toolchainDefault)
		if toolchain != "" {
			ctx.AddVariationDependencies(nil, tag.ToolchainTag, toolchain)
		}
	}

	if t, ok := ctx.Module().(*ModuleTest); ok {
		ctx.AddVariationDependencies(nil, tag.DepTag, t.Properties.Deps...)
		toolchain := proptools.StringDefault(t.Properties.Toolchain, toolchainDefault)
		if toolchain != "" {
			ctx.AddVariationDependencies(nil, tag.ToolchainTag, toolchain)
		}
	}

	if km, ok := ctx.Module().(*ModuleKernelObject); ok {
		ctx.AddDependency(ctx.Module(), tag.KernelModuleTag, km.Properties.Extra_symbols...)
	}

	if ins, ok := ctx.Module().(installable); ok {
		props := ins.getInstallableProps()
		if props.Install_group != nil {
			ctx.AddDependency(ctx.Module(), tag.InstallGroupTag, proptools.String(props.Install_group))
		}
		parseAndAddVariationDeps(ctx, tag.InstallTag, props.Install_deps...)
	}
	if strlib, ok := ctx.Module().(stripable); ok {
		info := strlib.getDebugInfo()
		if info != nil {
			ctx.AddDependency(ctx.Module(), tag.DebugInfoTag, *info)
		}
	}
}

func ResolveGenericDepsMutator(ctx blueprint.BottomUpMutatorContext) {
	ctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(dep) == tag.DepTag
		},
		func(dep blueprint.Module) {

			switch dep.(type) {
			case *ModuleStaticLibrary:
				ctx.AddVariationDependencies(nil, tag.StaticTag, dep.Name())
			case *ModuleSharedLibrary:
				ctx.AddVariationDependencies(nil, tag.SharedTag, dep.Name())
			case *ModuleStrictLibrary:
				lib := dep.(*ModuleStrictLibrary)

				if proptools.Bool(lib.Properties.Alwayslink) &&
					proptools.Bool(lib.Properties.Linkstatic) {
					ctx.AddVariationDependencies(nil, tag.WholeStaticTag, dep.Name())
				} else if proptools.Bool(lib.Properties.Linkstatic) {
					ctx.AddVariationDependencies(nil, tag.StaticTag, dep.Name())
				} else {
					ctx.AddVariationDependencies(nil, tag.SharedTag, dep.Name())
				}
				// TODO: implement tag.HeaderTag
			}
		})
}

// Applies target specific properties within each module. Must be done
// after the libraries have been split.
func targetMutator(ctx blueprint.TopDownMutatorContext) {
	if t, ok := ctx.Module().(targetSpecificLibrary); ok {

		tgt := t.getTarget()

		if tgt != toolchain.TgtTypeHost && tgt != toolchain.TgtTypeTarget {
			// This is fine if target is neither host or target,
			// it can happen if the target is the default
			return
		}

		dst := t.targetableProperties()
		src := t.getTargetSpecific(tgt).getTargetSpecificProps()

		// Copy the target-specific variables to the core set
		err := AppendMatchingProperties(dst, src)
		if err != nil {
			if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
				ctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
			} else {
				panic(err)
			}
		}
	}
}

type pathProcessor interface {
	// Prepares any path attributes, in most cases this means prefixing the module path to make sources
	// relative to Bob root directory.
	// This mutator should **only** modify paths, no other work should be done here.
	processPaths(blueprint.BaseModuleContext)
}

// Adds module paths to appropriate properties.
func pathMutator(ctx blueprint.BottomUpMutatorContext) {
	if p, ok := ctx.Module().(pathProcessor); ok {
		p.processPaths(ctx)
	}
}

func collectReexportLibsDependenciesMutator(ctx blueprint.TopDownMutatorContext) {
	mainModule := ctx.Module()
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

	ctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		depTag := ctx.OtherModuleDependencyTag(child)
		recurse := false

		if depTag == tag.WholeStaticTag || depTag == tag.StaticTag || depTag == tag.SharedTag {
			parentModule, ok1 := parent.(moduleWithBuildProps)
			childModule, ok2 := child.(moduleWithBuildProps)

			if !ok1 || !ok2 {
				return recurse
			}

			parentBuild := parentModule.build()
			childBuild := childModule.build()

			if len(childBuild.Reexport_libs) > 0 &&
				(parent.Name() == mainModule.Name() || utils.Contains(parentBuild.Reexport_libs, child.Name())) {
				mainBuild.ResolvedReexportedLibs = utils.AppendUnique(mainBuild.ResolvedReexportedLibs, childBuild.Reexport_libs)
				recurse = true
			}

			// Export_generated_headers works  exactly the same as Reexport_libs except for generated headers via genrules.
			if len(childBuild.Export_generated_headers) > 0 &&
				(parent.Name() == mainModule.Name() || utils.Contains(parentBuild.Export_generated_headers, child.Name())) {
				mainBuild.ResolvedGeneratedHeaders = utils.AppendUnique(mainBuild.ResolvedGeneratedHeaders, childBuild.Export_generated_headers)
				recurse = true
			}

			return recurse
		}

		return recurse
	})

}

func applyReexportLibsDependenciesMutator(ctx blueprint.BottomUpMutatorContext) {
	mainModule := ctx.Module()
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	var build *Build
	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		build = buildProps.build()
		ctx.AddVariationDependencies(nil, tag.ReexportLibraryTag, build.ResolvedReexportedLibs...)
		// Does not use variants as the resolved providers are not target aware (source generators)
		ctx.AddDependency(mainModule, tag.ReexportLibraryTag, build.ResolvedGeneratedHeaders...)
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
	} else if _, ok := ctx.Module().(*ModuleDefaults); ok { // Ignore defaults.
		return
	} else if _, ok := ctx.Module().(*ModuleAlias); ok { // Ignore aliases.
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

func checkDisabledMutator(ctx blueprint.BottomUpMutatorContext) {
	module := ctx.Module()
	// Skip if already disabled, or if defaults type,
	// or if type is not enableable (eg. alias)
	ep, ok := module.(enableable)
	if ok {
		if _, ok := module.(*ModuleDefaults); ok {
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

	ctx.VisitDirectDeps(func(dep blueprint.Module) {
		// ignore defaults - it's allowed for them to be disabled
		if _, ok := dep.(*ModuleDefaults); ok {
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
			utils.Die("Module %s is required but depends on disabled modules %s", module.Name(), strings.Join(disabledDeps, ", "))
		} else {
			ep.getEnableableProps().Enabled = proptools.BoolPtr(false)
			return
		}
	}
}

type FactoryWithConfig func(*BobConfig) (blueprint.Module, []interface{})

func RegisterModuleTypes(register func(string, FactoryWithConfig)) {
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

	// Swapping to new rules that are more strict and adhere to the Android Modules
	register("bob_genrule", generateRuleAndroidFactory)
	register("bob_gensrcs", gensrcsFactory)
	register("bob_filegroup", filegroupFactory)
	register("bob_glob", globFactory)
	register("bob_library", LibraryFactory)
	register("bob_executable", StrictBinaryFactory)

	register("bob_alias", aliasFactory)
	register("bob_kernel_module", kernelModuleFactory)
	register("bob_resource", resourceFactory)
	register("bob_install_group", installGroupFactory)

	register("bob_toolchain", ModuleToolchainFactory)
	register("bob_test", executableTestFactory)
}
