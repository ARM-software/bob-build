/*
 * Copyright 2023 Arm Limited.
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

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

func propogateLibraryDefinesMutator(ctx blueprint.BottomUpMutatorContext) {
	accumlatedDeps := []string{}
	ctx.VisitDirectDeps(func(dep blueprint.Module) {
		if strictLib, ok := dep.(*ModuleStrictLibrary); ok {
			accumlatedDeps = append(accumlatedDeps, strictLib.Properties.Deps...)
		}
	})

	if l, ok := ctx.Module().(*ModuleStrictLibrary); ok {
		l.Properties.Deps = append(l.Properties.Deps, accumlatedDeps...)
		ctx.AddDependency(ctx.Module(), StaticTag, accumlatedDeps...)
	} else if l, ok := getLibrary(ctx.Module()); ok {
		l.Properties.Static_libs = append(l.Properties.Static_libs, accumlatedDeps...)
		ctx.AddVariationDependencies(nil, StaticTag, accumlatedDeps...)
	}
}

func (g *linuxGenerator) strictLibraryStaticActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext, objectFiles []string) {
	m.Static.outputdir = backend.Get().StaticLibOutputDir(m.Properties.TargetType)
	m.Static.outs = []string{filepath.Join(m.Static.outputDir(), m.Name()+".a")}

	tc := backend.Get().GetToolchain(m.Properties.TargetType)
	arBinary, _ := tc.GetArchiver()

	depfiles := []string{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(m) == StaticTag
		},
		func(m blueprint.Module) {
			gen, _ := m.(*ModuleStrictLibrary)
			depfiles = append(depfiles, gen.Static.outputs()...)
		})
	args := map[string]string{
		"ar": arBinary,
	}
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:      staticLibraryRule,
			Outputs:   m.Static.outputs(),
			Inputs:    append(objectFiles),
			OrderOnly: depfiles,
			Optional:  true,
			Args:      args,
		})

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:    blueprint.Phony,
			Inputs:  m.Static.outputs(),
			Outputs: []string{m.shortName() + ".a"},
		})
}

func (g *linuxGenerator) strictLibrarySharedActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext, objectFiles []string) {
	m.Shared.outputdir = backend.Get().SharedLibsDir(m.Properties.TargetType)
	soFile := filepath.Join(m.Shared.outputDir(), m.Name()+".so")
	m.Shared.outs = []string{soFile}

	//TODO: Do we need symlink rules?

	// // Create symlinks if needed
	// for name, symlinkTgt := range m.librarySymlinks(ctx) {
	// 	symlink := filepath.Join(m.outputDir(), name)
	// 	lib := filepath.Join(m.outputDir(), symlinkTgt)
	// 	ctx.Build(pctx,
	// 		blueprint.BuildParams{
	// 			Rule:     symlinkRule,
	// 			Inputs:   []string{lib},
	// 			Outputs:  []string{symlink},
	// 			Args:     map[string]string{"target": symlinkTgt},
	// 			Optional: true,
	// 		})
	// 	installDeps = append(installDeps, symlink)
	// }

	// orderOnly := buildWrapperDeps
	// if enableToc {
	// 	// Add an order only dependecy on the actual libraries to cover
	// 	// the case where the .so is deleted but the toc is still
	// 	// present.
	// 	orderOnly = append(orderOnly, g.getSharedLibLinkPaths(ctx)...)
	// }

	tc := backend.Get().GetToolchain(m.Properties.TargetType)
	linker := tc.GetLinker().GetTool()
	args := map[string]string{
		"linker":          linker,
		"shared_libs_dir": m.Shared.outputdir,
	}

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     sharedLibraryRule,
			Outputs:  m.Shared.outputs(),
			Inputs:   objectFiles,
			Optional: true,
			Args:     args,
		})

	g.addSharedLibToc(ctx, soFile, m.Shared.outputDir()+"/"+m.Name()+".toc", m.getTarget())

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:    blueprint.Phony,
			Inputs:  m.Shared.outputs(),
			Outputs: []string{m.shortName() + ".so"},
		})
}

func (g *linuxGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objectFiles, _ := CompileObjs(m, ctx, tc)

	g.strictLibraryStaticActions(m, ctx, objectFiles)
	// TODO: Stub the shared lib implementation and break it off of this patch.
	// g.strictLibrarySharedActions(m, ctx, objectFiles)
}

func proxyCflags(m *ModuleStrictLibrary) []string {
	Cflags := m.Properties.Copts
	for _, def := range m.Properties.Local_defines {
		Cflags = append(Cflags, "-D"+def)
	}
	for _, def := range m.Properties.Defines {
		Cflags = append(Cflags, "-D"+def)
	}
	return Cflags
}

func (g *androidBpGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	// TODO: Move this to it's own file

	// TODO: Handle shared library versions too
	var proxyStaticLib ModuleStaticLibrary
	proxyStaticLib.SimpleName.Properties.Name = m.SimpleName.Properties.Name
	proxyStaticLib.Properties.EnableableProps.Required = true
	proxyStaticLib.Properties.Srcs = m.Properties.Srcs
	proxyStaticLib.Properties.Cflags = proxyCflags(m)
	proxyStaticLib.Properties.Host_supported = m.Properties.Host_supported
	proxyStaticLib.Properties.Target_supported = m.Properties.Target_supported
	// TODO: generate target for all supported target types
	proxyStaticLib.Properties.TargetType = toolchain.TgtTypeHost

	proxyStaticLib.Properties.ResolveFiles(ctx)
	g.staticActions(&proxyStaticLib, ctx)
	// TODO: Static lib dependency
}
