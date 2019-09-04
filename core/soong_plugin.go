// +build soong

/*
 * Copyright 2019 Arm Limited.
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

/*
 * This file is included when Bob is being run as a Soong plugin.
 *
 * The build tag on the first line ensures that it is not included in the build
 * by accident, and that it is not included in `go test` or similar checks,
 * which would fail, because Soong is not available in that environment.
 */

package core

import (
	"path/filepath"
	"sync"

	"android/soong/android"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/abstr"
	"github.com/ARM-software/bob-build/graph"
)

const (
	bobModuleSuffix = "__bob_module_type"
)

var (
	loadConfigOnce   sync.Once
	onceLoadedConfig *bobConfig
)

// During the build, Soong will do a "test" of each plugin, which loads the
// module, including calling its `init()` functions. That means that we can't
// load the config file in `init()`, because the tests would fail if it doesn't
// exist. Work around this by deferring loading the config file until a module
// factory is actually called.
func soongGetConfig() *bobConfig {
	loadConfigOnce.Do(func() {
		onceLoadedConfig = &bobConfig{}
		onceLoadedConfig.Properties = loadConfig(jsonPath)

		if !onceLoadedConfig.Properties.GetBool("builder_soong") {
			panic("Build bootstrapped for Soong, but Soong builder has not been enabled")
		}
		onceLoadedConfig.Generator = &soongGenerator{}
	})
	return onceLoadedConfig

}

func getConfig(interface{}) *bobConfig {
	return soongGetConfig()
}

type moduleBase struct {
	// android.ModuleBase and blueprint.SimpleName both contain `Name`
	// properties and methods. However, we can't access the one provided by
	// android.ModuleBase without calling InitAndroidModule(), which would
	// also add a load of other properties that we don't want. So embed
	// SimpleName here, and provide a Name() method to choose which one to
	// delegate to.
	blueprint.SimpleName

	android.ModuleBase
}

func (m *moduleBase) GenerateAndroidBuildActions(ctx android.ModuleContext) {}

func (m *moduleBase) Name() string { return m.SimpleName.Name() }

type soongGenerator struct {
	toolchainSet
}

func (g *soongGenerator) aliasActions(m *alias, ctx blueprint.ModuleContext)                        {}
func (g *soongGenerator) binaryActions(*binary, blueprint.ModuleContext)                            {}
func (g *soongGenerator) genBinaryActions(*generateBinary, blueprint.ModuleContext, []inout)        {}
func (g *soongGenerator) genSharedActions(*generateSharedLibrary, blueprint.ModuleContext, []inout) {}
func (g *soongGenerator) genStaticActions(*generateStaticLibrary, blueprint.ModuleContext, []inout) {}
func (g *soongGenerator) generateSourceActions(*generateSource, blueprint.ModuleContext, []inout)   {}
func (g *soongGenerator) kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext)          {}
func (g *soongGenerator) resourceActions(*resource, blueprint.ModuleContext)                        {}
func (g *soongGenerator) sharedActions(*sharedLibrary, blueprint.ModuleContext)                     {}
func (g *soongGenerator) staticActions(*staticLibrary, blueprint.ModuleContext)                     {}
func (g *soongGenerator) transformSourceActions(*transformSource, blueprint.ModuleContext, []inout) {}

func (g *soongGenerator) buildDir() string                           { return "" }
func (g *soongGenerator) sourcePrefix() string                       { return "" }
func (g *soongGenerator) sharedLibsDir(tgt tgtType) string           { return "" }
func (g *soongGenerator) sourceOutputDir(m *generateCommon) string   { return "" }
func (g *soongGenerator) binaryOutputDir(m *binary) string           { return "" }
func (g *soongGenerator) staticLibOutputDir(m *staticLibrary) string { return "" }
func (g *soongGenerator) sharedLibOutputDir(m *sharedLibrary) string { return "" }
func (g *soongGenerator) kernelModOutputDir(m *kernelModule) string  { return "" }

func (g *soongGenerator) init(*blueprint.Context, *bobConfig) {}

// Bob modules that need Soong to run LoadHooks need to implement this
// interface.
type soongBuildActionsProvider interface {
	soongBuildActions(android.TopDownMutatorContext)
}

// Avoid conflicts with the Soong modules we generate by renaming the Bob
// modules at the last minute. Calls to `mctx.ModuleName()` will return the
// new name, but the module's `Name()` method will be unchanged.
//
// Unfortunately we can't just do this right before calling CreateModule,
// because renames are only enacted after each mutator pass. Therefore it is
// done it its own mutator, before buildActionsMutator.
func renameMutator(mctx android.TopDownMutatorContext) {
	if _, ok := mctx.Module().(soongBuildActionsProvider); !ok {
		return
	}

	mctx.Rename(mctx.ModuleName() + bobModuleSuffix)
}

func buildActionsMutator(mctx android.TopDownMutatorContext) {
	m, ok := mctx.Module().(soongBuildActionsProvider)
	if !ok {
		return
	}

	m.soongBuildActions(mctx)
}

func registerMutators(ctx android.RegisterMutatorsContext) {
	ctx.BottomUp("bob_default_deps", abstr.BottomUpAdaptor(defaultDepsMutator)).Parallel()
	ctx.TopDown("bob_features_applier", abstr.TopDownAdaptor(featureApplierMutator)).Parallel()
	ctx.TopDown("bob_template_applier", abstr.TopDownAdaptor(templateApplierMutator)).Parallel()
	ctx.BottomUp("bob_check_lib_fields", abstr.BottomUpAdaptor(checkLibraryFieldsMutator)).Parallel()
	ctx.BottomUp("bob_process_paths", abstr.BottomUpAdaptor(pathMutator)).Parallel()
	ctx.BottomUp("bob_strip_empty_components", abstr.BottomUpAdaptor(stripEmptyComponentsMutator)).Parallel()
	ctx.TopDown("bob_supported_variants", abstr.TopDownAdaptor(supportedVariantsMutator)).Parallel()
	ctx.BottomUp(splitterMutatorName, abstr.BottomUpAdaptor(splitterMutator)).Parallel()
	ctx.TopDown("bob_target", abstr.TopDownAdaptor(targetMutator)).Parallel()
	ctx.TopDown("bob_default_applier", abstr.TopDownAdaptor(defaultApplierMutator)).Parallel()
	ctx.BottomUp("bob_depender", abstr.BottomUpAdaptor(dependerMutator)).Parallel()
	ctx.BottomUp("bob_generated", abstr.BottomUpAdaptor(generatedDependerMutator)).Parallel()
	dependencyGraphHandler := graphMutatorHandler{graph.NewGraph("All")}
	ctx.BottomUp("bob_sort_resolved_static_libs",
		abstr.BottomUpAdaptor(dependencyGraphHandler.ResolveDependencySortMutator)) // This can't be parallel
	ctx.TopDown("bob_find_required_modules", abstr.TopDownAdaptor(findRequiredModulesMutator)).Parallel()
	ctx.TopDown("bob_check_reexport_libs", abstr.TopDownAdaptor(checkReexportLibsMutator)).Parallel()
	ctx.TopDown("bob_collect_reexport_lib_dependencies",
		abstr.TopDownAdaptor(collectReexportLibsDependenciesMutator)).Parallel()
	ctx.BottomUp("bob_apply_reexport_lib_dependencies",
		abstr.BottomUpAdaptor(applyReexportLibsDependenciesMutator)).Parallel()
	ctx.TopDown("bob_rename", renameMutator).Parallel()
	ctx.TopDown("bob_build_actions", buildActionsMutator).Parallel()
}

func soongRegisterModule(name string, mf factoryWithConfig) {
	// Create a closure adapting Bob's module factories to the format Soong uses.
	factory := func() android.Module {
		bpModule, properties := mf(soongGetConfig())
		// This type assertion should always pass as long as every Bob
		// module type embeds moduleBase
		soongModule := bpModule.(android.Module)

		for _, property := range properties {
			soongModule.AddProperties(property)
		}

		return soongModule
	}
	android.RegisterModuleType(name, factory)
}

func init() {
	registerModuleTypes(soongRegisterModule)

	android.PreArchMutators(registerMutators)
}

// Some module types generate other Soong modules. For these, the sources must
// be specified relative to the original module's subdirectory. This helper
// calculates this, effectively undoing most of the work of the process_paths
// mutator.
func relativeToModuleDir(mctx android.BaseModuleContext, paths []string) (srcs []string) {
	for _, path := range paths {
		rel, err := filepath.Rel(mctx.ModuleDir(), path)
		if err != nil {
			panic(err)
		}
		srcs = append(srcs, rel)
	}
	return
}
