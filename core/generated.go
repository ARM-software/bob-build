package core

import (
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
)

// Returns the outputs of the generated dependencies of a module. This is used for more complex
// dependencies, where the dependencies are not just binaries or headers, but where the paths are
// used directly in a script
func getDependentArgsAndFiles(ctx blueprint.ModuleContext, args map[string]string) (depfiles []string, fullDeps map[string][]string) {
	fullDeps = make(map[string][]string)
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(m) == tag.GeneratedTag
		},
		func(m blueprint.Module) {

			// Dependent `Tools` which are `ModuleFilegroup`
			if fg, ok := m.(*ModuleFilegroup); ok {
				var buildPaths []string

				fg.OutFiles().ForEach(
					func(fp file.Path) bool {
						buildPaths = append(buildPaths, fp.BuildPath())
						return true
					})

				depfiles = append(depfiles, buildPaths...)
				fullDeps[fg.shortName()] = buildPaths
				return
			}

			gen, ok := m.(dependentInterface)
			if !ok {
				utils.Die("%s is not a valid dependent interface", reflect.TypeOf(m).String())
			}

			depName := ctx.OtherModuleName(m)
			// When the dependent module is another Bob generated
			// module, provide all its outputs so the using module can
			// pick and choose what it uses.
			args[depName+"_out"] = strings.Join(gen.outputs(), " ")

			depfiles = append(depfiles, gen.outputs()...)
			depfiles = append(depfiles, file.GetImplicitOutputs(gen)...)

			fullDeps[gen.shortName()] = depfiles
		})

	return
}

func getDepfileName(s string) string {
	return utils.FlattenPath(s) + ".d"
}

func getRspfileName(s string) string {
	return "." + utils.FlattenPath(s) + ".rsp"
}

// ModuleContext Helpers

// Return the outputs() of all GeneratedSource dependencies of the current
// module. The current module can be generated or a library, and the
// dependencies can be anything implementing DependentInterface (so "generated"
// is a misnomer, because this includes libraries, too).
func getGeneratedFiles(ctx blueprint.ModuleContext) []string {
	var srcs []string
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.GeneratedSourcesTag },
		func(m blueprint.Module) {
			if gs, ok := m.(dependentInterface); ok {
				srcs = append(srcs, gs.outputs()...)
				srcs = append(srcs, gs.implicitOutputs()...)
			} else {
				utils.Die("%s does not have outputs", ctx.OtherModuleName(m))
			}
		})
	return srcs
}

func generatedDependerMutator(ctx blueprint.BottomUpMutatorContext) {

	if _, ok := ctx.Module().(*ModuleGenerateSource); ok {
		backend.Get().GetLogger().Warn(warnings.GenerateRuleWarning, ctx.BlueprintsFile(), ctx.ModuleName())
	}

	if e, ok := ctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	// Things which depend on generated/transformed sources
	if l, ok := getLibrary(ctx.Module()); ok {
		ctx.AddDependency(ctx.Module(), tag.GeneratedSourcesTag, l.Properties.Generated_sources...)
		ctx.AddDependency(ctx.Module(), tag.GeneratedHeadersTag, l.Properties.Generated_headers...)
		ctx.AddDependency(ctx.Module(), tag.ExportGeneratedHeadersTag, l.Properties.Export_generated_headers...)
		ctx.AddDependency(ctx.Module(), tag.GeneratedTag, l.Properties.Generated_deps...)
	}

	// Things that a generated/transformed source depends on
	if gsc, ok := getGenerateCommon(ctx.Module()); ok {
		if gsc.Properties.Host_bin != nil {
			parseAndAddVariationDeps(ctx, tag.HostToolBinaryTag,
				proptools.String(gsc.Properties.Host_bin))
		}
		// Generated sources can use the outputs of another generated
		// source or library as a source file or dependency.
		parseAndAddVariationDeps(ctx, tag.GeneratedTag,
			gsc.Properties.Generated_deps...)
		parseAndAddVariationDeps(ctx, tag.GeneratedSourcesTag,
			gsc.Properties.Generated_sources...)

		for _, d := range gsc.deps {
			// Add other module dependency
			ctx.AddDependency(ctx.Module(), tag.GeneratedTag, d)
		}
	}

	if _, ok := getGenerator(ctx).(*linuxGenerator); ok {
		if agsc, ok := getStrictGenerateCommon(ctx.Module()); ok {
			for _, s := range agsc.Properties.Srcs {
				if s[0] == ':' {
					parseAndAddVariationDeps(ctx, tag.GeneratedSourcesTag,
						s[1:])
					parseAndAddVariationDeps(ctx, tag.GeneratedTag,
						s[1:])
				}
			}

			for _, d := range agsc.deps {
				// Add other module dependency
				ctx.AddDependency(ctx.Module(), tag.GeneratedTag, d)
			}
		}
	}

	// For strict generation rules, i.e. `bob_genule` & `bob_gensrcs`, the `tool` property
	// is for the modules that produces the host executable. Thus those should follow with
	// `tag.HostToolBinaryTag` tag dependency.
	if agsc, ok := getStrictGenerateCommon((ctx.Module())); ok {
		parseAndAddVariationDeps(ctx, tag.HostToolBinaryTag, agsc.Properties.Tools...)
	}
}

// hostBinOuts returns the tool binary ('host_bin') together with its
// target type and shared library dependencies for a generator module.
// This is different from the "tool" in that it used to depend on
// a bob_binary module.
func hostBinOuts(hostBin *string, ctx blueprint.ModuleContext) (string, []string, toolchain.TgtType) {
	// No host_bin provided
	if hostBin == nil {
		return "", []string{}, toolchain.TgtTypeUnknown
	}

	hostBinOut := ""
	hostBinSharedLibsDeps := []string{}
	hostBinTarget := toolchain.TgtTypeUnknown
	hostBinFound := false

	ctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		depTag := ctx.OtherModuleDependencyTag(child)

		if parent == ctx.Module() && depTag == tag.HostToolBinaryTag {
			var outputs []string
			hostBinFound = true

			if b, ok := child.(*ModuleBinary); ok {
				outputs = b.outputs()
				hostBinTarget = b.getTarget()
			} else if gb, ok := child.(*generateBinary); ok {
				outputs = gb.outputs()
			} else {
				ctx.PropertyErrorf("host_bin", "%s is not a `bob_binary` nor `bob_generate_binary`", parent.Name())
				return false
			}

			if len(outputs) != 1 {
				ctx.OtherModuleErrorf(child, "outputs() returned %d outputs", len(outputs))
			} else {
				hostBinOut = outputs[0]
			}

			return true // keep visiting
		} else if parent != ctx.Module() && depTag == tag.SharedTag {
			if l, ok := child.(*ModuleSharedLibrary); ok {
				hostBinSharedLibsDeps = append(hostBinSharedLibsDeps, l.outputs()...)
			}

			return true // keep visiting
		} else {
			return false // stop visiting
		}
	})

	if !hostBinFound {
		ctx.ModuleErrorf("Could not find module specified by `host_bin: %v`", hostBin)
	}

	return hostBinOut, hostBinSharedLibsDeps, hostBinTarget
}
