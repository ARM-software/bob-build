/*
 * This file is included when Bob is being run as a standalone binary, i.e. for
 * the Ninja generator.
 */

package core

import (
	"os"

	"github.com/google/blueprint"
	"github.com/google/blueprint/bootstrap"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/graph"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
)

type PropertyProvider interface {
	GetProperties() interface{}
}

// configProvider allows the retrieval of configuration
type configProvider interface {
	Config() interface{}
}

func projectModuleDir(ctx blueprint.BaseModuleContext) string {
	return ctx.ModuleDir()
}

func getConfig(ctx configProvider) *BobConfig {
	return ctx.Config().(*BobConfig)
}

func getBuildDir() string {
	if bootstrap.BuildDir == "" {
		panic("bootstrap.BuildDir was not initialized!")
	}
	return bootstrap.BuildDir
}

func getSourceDir() string {
	// TODO: This should be part of the backend.
	return config.GetEnvironmentVariables().SrcDir
}

func getBobDir() string {
	// TODO: This should be part of the backend.
	return config.GetEnvironmentVariables().BobDir
}

// Main is the entry point for the bob primary builder.
//
// It loads the configuration from .bob.config.json, registers the module type
// and mutators, initializes the backend, and finally calls into Blueprint.
func Main() {
	// Load the config first. This is needed because some of the module
	// types' definitions contain a struct-per-feature, and features are
	// specified in the cfg.
	cfg := &BobConfig{}
	env := config.GetEnvironmentVariables()

	err := cfg.Properties.LoadConfig(env.ConfigJSON)
	if err != nil {
		utils.Die("%v", err)
	}

	builder_ninja := cfg.Properties.GetBool("builder_ninja")
	builder_android_bp := cfg.Properties.GetBool("builder_android_bp")

	// Depend on the config file
	pctx.AddNinjaFileDeps(env.ConfigJSON, getPathInBuildDir(".env.hash"))

	var ctx = blueprint.NewContext()

	RegisterModuleTypes(func(name string, mf FactoryWithConfig) {
		// Create a closure passing the config to a module factory so
		// that the module factories can access the cfg.
		factory := func() (blueprint.Module, []interface{}) {
			return mf(cfg)
		}
		ctx.RegisterModuleType(name, factory)
	})

	// Note that the order of mutators is important, since the
	// contents of each module will be rewritten. The following
	// describes the required orderring of mutators dealing with
	// property propagation.
	//
	// On reading build.bp, the various properties will be set
	// according to the build.bp structure:
	//
	//  .props.propA
	//  .props.feature1.propA
	//  .Host.props.propA
	//  .Host.props.feature1.propA
	//  .Target.props.propA
	//  .Target.props.feature1.propA
	//
	//  default.props.propA
	//  default.props.feature1.propA
	//  default.Host.props.propA
	//  default.Host.props.feature1.propA
	//  default.Target.props.propA
	//  default.Target.props.feature1.propA
	//
	// Merge feature-specific values to the level above in each
	// module. This must be before defaults so that a feature-specific
	// option set in a default does not override an option set in a
	// module. Do this before templates so templates only need to
	// operate on one level. The properties we care about are then:
	//
	//  .props.propA
	//  .Host.props.propA
	//  .Target.props.propA
	//
	//  default.props.propA
	//  default.Host.props.propA
	//  default.Target.props.propA
	//
	// Evaluate templates next, including in defaults. This avoids us
	// having to re-evaluate templates after they have been copied
	// around by defaults.
	//
	// The supported_variants mutator runs next. This just propagates the
	// host_supported and target_supported properties through the
	// defaults, allowing us to identify whether each module supports
	// host and target, and split the modules early.
	//
	// Then split the libraries into host-specific and target-specific
	// modules.
	//
	// After the libraries are split we can apply target-specific
	// options, flattening the properties further:
	//
	//  .props.propA
	//
	//  default.props.propA
	//
	// Finally apply the remaining properties from defaults. This
	// leaves the main property structure on each module holding all
	// the settings for each property:
	//
	//  .props.propA
	//
	// The depender mutator adds the dependencies between binaries and libraries.
	//
	// The generated depender mutator add dependencies to generated source modules.
	ctx.RegisterEarlyMutator("register_toolchains", RegisterToolchainModules)
	ctx.RegisterBottomUpMutator("default_deps1", DefaultDepsStage1Mutator).Parallel()
	ctx.RegisterBottomUpMutator("default_deps2", DefaultDepsStage2Mutator).Parallel()
	ctx.RegisterTopDownMutator("features_applier", featureApplierMutator).Parallel()
	ctx.RegisterBottomUpMutator("check_lib_fields", checkLibraryFieldsMutator).Parallel()
	ctx.RegisterBottomUpMutator("check_genrule_fields", checkGenruleFieldsMutator).Parallel()
	ctx.RegisterBottomUpMutator("strip_empty_components", stripEmptyComponentsMutator).Parallel()
	ctx.RegisterBottomUpMutator("supported_variants", supportedVariantsMutator).Parallel()
	ctx.RegisterBottomUpMutator(splitterMutatorName, splitterMutator).Parallel()
	ctx.RegisterTopDownMutator("target", targetMutator).Parallel()

	// `pathMutator` has to run before `DefaultApplierMutator`. This is because paths declared in the module
	// are relative to their module scope, whereas paths declared in the defaults are not.
	ctx.RegisterBottomUpMutator("process_paths", pathMutator).Parallel()

	ctx.RegisterBottomUpMutator("default_applier", DefaultApplierMutator).Parallel()
	ctx.RegisterBottomUpMutator("depender", dependerMutator).Parallel()
	ctx.RegisterBottomUpMutator("generic_depender", ResolveGenericDepsMutator).Parallel()

	// First resolve providers which are not dependant on other modules.
	ctx.RegisterBottomUpMutator("resolve_files", file.ResolveFilesMutator).Parallel()

	// Now we can resolve remaining, dynamic file providers.
	ctx.RegisterBottomUpMutator("resolve_dynamic_src_outputs", resolveDynamicFileOutputs) // Cannot be parallel.

	ctx.RegisterBottomUpMutator("alias", aliasMutator).Parallel()
	ctx.RegisterBottomUpMutator("generated", generatedDependerMutator).Parallel()
	ctx.RegisterBottomUpMutator("collect_metadata", metaDataCollector).Parallel()

	if handler := initGrapvizHandler(); handler != nil {
		ctx.RegisterBottomUpMutator("graphviz_output", handler.graphvizMutator)
		// Singleton for stop tool and don't overwrite build.bp
		ctx.RegisterSingletonType("quit_singleton", handler.quitSingletonFactory)
	} else {

		ctx.RegisterTopDownMutator("export_lib_flags", exportLibFlagsMutator).Parallel()
		dependencyGraphHandler := graphMutatorHandler{
			map[toolchain.TgtType]graph.Graph{
				toolchain.TgtTypeHost:   graph.NewGraph("All"),
				toolchain.TgtTypeTarget: graph.NewGraph("All"),
			},
		}
		ctx.RegisterBottomUpMutator("sort_resolved_static_libs",
			dependencyGraphHandler.ResolveDependencySortMutator) // This can't be parallel
		ctx.RegisterTopDownMutator("find_required_modules",
			findRequiredModulesMutator).Parallel()
		ctx.RegisterBottomUpMutator("check_disabled_modules",
			checkDisabledMutator).Parallel()
		ctx.RegisterTopDownMutator("check_reexport_libs",
			checkReexportLibsMutator).Parallel()
		ctx.RegisterTopDownMutator("collect_reexport_lib_dependencies",
			collectReexportLibsDependenciesMutator).Parallel()
		ctx.RegisterBottomUpMutator("apply_reexport_lib_dependencies",
			applyReexportLibsDependenciesMutator).Parallel()
		ctx.RegisterTopDownMutator("install_group_mutator", installGroupMutator).Parallel()
		ctx.RegisterTopDownMutator("debug_info_mutator", debugInfoMutator).Parallel()
		if !builder_android_bp {
			// The android_bp backend's escape function is a no-op,
			// so optimize by skipping the mutator
			ctx.RegisterTopDownMutator("escape_mutator", escapeMutator).Parallel()
		}
		ctx.RegisterTopDownMutator("late_template_mutator", lateTemplateMutator).Parallel()
	}

	f, err := os.OpenFile(env.LogWarningsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		utils.Die("error opening '%s' file: %v", env.LogWarningsFile, err)
	}

	logger := warnings.New(f, env.LogWarnings)

	defer func() {
		errCnt := logger.ErrorWarnings()
		f.Close()

		if errCnt > 0 {
			utils.Die("%d error(s) ocurred!\n\n%s\n", errCnt, logger.InfoMessage())
		}
	}()

	defer MetaDataWriteToFile(env.BuildMetaFile)

	if builder_ninja {
		cfg.Generator = &linuxGenerator{}
	} else if builder_android_bp {
		cfg.Generator = &androidBpGenerator{}

		// Do not run in parallel to avoid locking issues on the map
		ctx.RegisterBottomUpMutator("collect_buildbp", collectBuildBpFilesMutator)
		ctx.RegisterSingletonType("androidbp_singleton", androidBpSingletonFactory)
	} else {
		utils.Die("Unknown builder backend")
	}

	// It is safe to call `backend.Get()` after this call.
	backend.Setup(env,
		&cfg.Properties,
		logger,
	)

	bootstrap.Main(ctx, cfg)
}
