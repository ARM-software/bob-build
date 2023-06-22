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

func (g *linuxGenerator) strictLibraryActions(m *ModuleStrictLibrary, ctx blueprint.ModuleContext) {
	tc := backend.Get().GetToolchain(m.Properties.TargetType)

	objectFiles, _ := CompileObjs(m, ctx, tc)

	g.ArchivableActions(ctx, m, tc, objectFiles)
	// TODO: implement shared library outputs
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
