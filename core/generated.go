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
	"reflect"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
)

var (
	generatedHeaderTag       = DependencyTag{name: "generated_headers"}
	exportGeneratedHeaderTag = DependencyTag{name: "export_generated_headers"}
	generatedSourceTag       = DependencyTag{name: "generated_sources"}
	generatedDepTag          = DependencyTag{name: "generated_dep"}
	hostToolBinTag           = DependencyTag{name: "host_tool_bin"}
	filegroupTag             = DependencyTag{name: "filegroup"}
	implicitSrcsTag          = DependencyTag{name: "implicit_srcs"}
)

// Return a list of headers generated by this module with full paths
func getHeadersGenerated(m dependentInterface) []string {
	return append(m.outputs(), m.implicitOutputs()...)
}

// Return a list of source files (not headers) generated by this module with full paths
func getSourcesGenerated(m dependentInterface) []string {
	return append(m.outputs(), m.implicitOutputs()...)
}

// Returns the outputs of the generated dependencies of a module. This is used for more complex
// dependencies, where the dependencies are not just binaries or headers, but where the paths are
// used directly in a script
func getDependentArgsAndFiles(ctx blueprint.ModuleContext, args map[string]string) (depfiles []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(m) == generatedDepTag
		},
		func(m blueprint.Module) {
			gen, ok := m.(dependentInterface)
			if !ok {
				utils.Die("%s is not a valid dependent interface", reflect.TypeOf(m).String())
			}

			depName := ctx.OtherModuleName(m)
			// When the dependent module is another Bob generated
			// module, provide all its outputs so the using module can
			// pick and choose what it uses.
			if gc, ok := getGenerateCommon(m); ok {
				args[depName+"_out"] = strings.Join(gc.outputs(), " ")
			} else {
				args[depName+"_out"] = strings.Join(gen.outputs(), " ")
			}

			depfiles = append(depfiles, gen.outputs()...)
			depfiles = append(depfiles, gen.implicitOutputs()...)
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
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == generatedSourceTag },
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
		getBackend(ctx).getLogger().Warn(warnings.GenerateRuleWarning, ctx.BlueprintsFile(), ctx.ModuleName())
	}

	if e, ok := ctx.Module().(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, so don't add dependencies
			return
		}
	}

	// Things which depend on generated/transformed sources
	if l, ok := getLibrary(ctx.Module()); ok {
		ctx.AddDependency(ctx.Module(), generatedSourceTag, l.Properties.Generated_sources...)
		ctx.AddDependency(ctx.Module(), generatedHeaderTag, l.Properties.Generated_headers...)
		ctx.AddDependency(ctx.Module(), exportGeneratedHeaderTag, l.Properties.Export_generated_headers...)
		ctx.AddDependency(ctx.Module(), generatedDepTag, l.Properties.Generated_deps...)
	}

	// Things that a generated/transformed source depends on
	if gsc, ok := getGenerateCommon(ctx.Module()); ok {
		if gsc.Properties.Host_bin != nil {
			parseAndAddVariationDeps(ctx, hostToolBinTag,
				proptools.String(gsc.Properties.Host_bin))
		}
		// Generated sources can use the outputs of another generated
		// source or library as a source file or dependency.
		parseAndAddVariationDeps(ctx, generatedDepTag,
			gsc.Properties.Generated_deps...)
		parseAndAddVariationDeps(ctx, generatedSourceTag,
			gsc.Properties.Generated_sources...)
	}

	if _, ok := getBackend(ctx).(*linuxGenerator); ok {
		if agsc, ok := getAndroidGenerateCommon(ctx.Module()); ok {
			for _, s := range agsc.Properties.Srcs {
				if s[0] == ':' {
					parseAndAddVariationDeps(ctx, generatedSourceTag,
						s[1:])
					parseAndAddVariationDeps(ctx, generatedDepTag,
						s[1:])
				}
			}
		}
	}

	// These rules also need to support variants when depending on tools. This strictly breaks android's genrule definition.
	// However, if a colon appears at the end of a module name with a text string, we assume there is a variant
	// called <module_name>__<variant_name> generated. Which bob currently does. This will fix behaviour on Android, to
	// ensure it works on Linux, the backend must see this as a generated_dep which is processing done in the linux backend.
	if agsc, ok := getAndroidGenerateCommon((ctx.Module())); ok {
		var removeList []string
		for _, s := range agsc.Properties.Tools {
			if strings.Contains(s, ":") {
				if _, ok := getBackend(ctx).(*linuxGenerator); ok {
					parseAndAddVariationDeps(ctx, generatedDepTag,
						s)
				} else {
					agsc.Properties.Tools = append(agsc.Properties.Tools, strings.Replace(s, ":", "__", 1))
					removeList = append(removeList, s)
				}
			}
		}
		for i := range removeList {
			agsc.Properties.Tools = append(agsc.Properties.Tools[:i], agsc.Properties.Tools[i+1:]...)
		}
	}
}
