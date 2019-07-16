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

/*
 * This file is included when Bob is being run as a standalone binary, i.e. for
 * the Ninja and Android Make generators.
 */

package core

import (
	"errors"
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"

	"github.com/ARM-software/bob-build/graph"
)

var (
	jsonPath = filepath.Join(builddir, "config.json")
)

type moduleBase struct {
	blueprint.SimpleName
}

// configProvider allows the retrieval of configuration
type configProvider interface {
	Config() interface{}
}

func getConfig(ctx configProvider) *bobConfig {
	return ctx.Config().(*bobConfig)
}

// Mutator to apply templates in standalone Bob
func templateApplierMutator(mctx blueprint.TopDownMutatorContext) {
	templateApplier(mctx.Module(), getConfig(mctx), mctx)
}

// Mutator to apply features in standalone Bob
func featureApplierMutator(mctx blueprint.TopDownMutatorContext) {
	featureApplier(mctx.Module(), getConfig(mctx), mctx)
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

	registerModuleTypes(func(name string, mf factoryWithConfig) {
		// Create a closure passing the config to a module factory so
		// that the module factories can access the config.
		factory := func() (blueprint.Module, []interface{}) {
			return mf(config)
		}
		ctx.RegisterModuleType(name, factory)
	})

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
	ctx.RegisterBottomUpMutator("process_build_wrapper", buildWrapperMutator).Parallel()
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

		ctx.RegisterTopDownMutator("check_reexport_libs", checkReexportLibsMutator).Parallel()
		ctx.RegisterTopDownMutator("collect_reexport_lib_dependencies", collectReexportLibsDependenciesMutator).Parallel()
		ctx.RegisterBottomUpMutator("apply_reexport_lib_dependencies", applyReexportLibsDependenciesMutator).Parallel()
		ctx.RegisterTopDownMutator("encapsulates_mutator", encapsulatesMutator).Parallel()
		ctx.RegisterTopDownMutator("install_group_mutator", installGroupMutator).Parallel()
		ctx.RegisterTopDownMutator("match_sources_mutator", matchSourcesMutator).Parallel()

		// Depend on the config file
		ctx.RegisterSingletonType("config_singleton", dependencySingletonFactory)
	}

	if config.Properties.GetBool("builder_ninja") {
		config.Generator = &linuxGenerator{}
	} else if config.Properties.GetBool("builder_android_make") {
		config.Generator = &androidMkGenerator{}
	} else {
		panic(errors.New("unknown builder backend"))
	}

	config.Generator.init(ctx, config)
	bootstrap.Main(ctx, config)
}
