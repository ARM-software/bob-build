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
	"path"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

func propogateLibraryDefinesMutator(mctx blueprint.BottomUpMutatorContext) {
	accumlatedDefines := []string{}
	accumlatedDeps := []string{}
	mctx.VisitDirectDeps(func(dep blueprint.Module) {
		strictLib, ok := dep.(*strictLibrary)
		if ok {
			for _, define := range strictLib.Properties.Defines {
				accumlatedDefines = append(accumlatedDefines, define)
			}
			for _, dep := range strictLib.Properties.Deps {
				accumlatedDeps = append(accumlatedDeps, dep)
			}
		}
		legacyLib, ok := getLibrary(dep)
		if ok {
			for _, define := range legacyLib.Properties.Defines {
				accumlatedDefines = append(accumlatedDefines, define)
			}
		}
	})

	if sl, ok := mctx.Module().(*strictLibrary); ok {
		sl.Properties.Defines = append(sl.Properties.Defines, accumlatedDefines...)
		sl.Properties.Deps = append(sl.Properties.Deps, accumlatedDeps...)
		mctx.AddDependency(mctx.Module(), staticDepTag, accumlatedDeps...)
	} else if l, ok := getLibrary(mctx.Module()); ok {
		for _, define := range accumlatedDefines {
			l.Properties.Cflags = append(l.Properties.Cflags, "-D"+define)
			l.Properties.Defines = append(l.Properties.Defines, define)
			// TODO: how we decide on static vs. shared?
			l.Properties.Static_libs = append(l.Properties.Static_libs, accumlatedDeps...)
			mctx.AddVariationDependencies(nil, staticDepTag, accumlatedDeps...)
		}
	}
}

func (l *strictLibrary) CompileObjs(ctx blueprint.ModuleContext) ([]string, []string) {
	g := getBackend(ctx)
	srcs := l.getSrcs()

	tc := g.getToolchain(l.Properties.TargetType)
	as, astargetflags := tc.getAssembler()
	cc, cctargetflags := tc.getCCompiler()
	cxx, cxxtargetflags := tc.getCXXCompiler()
	var cflagsList []string = nil
	for _, local_define := range l.Properties.Local_defines {
		cflagsList = append(cflagsList, ("-D" + local_define))
	}
	for _, local_define := range l.Properties.Defines {
		// TODO: For legacy libraries, this gets set in the mutator, it is unsymmetrical to set
		// this up here.
		cflagsList = append(cflagsList, ("-D" + local_define))
	}
	cflagsList = append(cflagsList, l.Properties.Copts...)

	ctx.Variable(pctx, "asflags", utils.Join(astargetflags))
	ctx.Variable(pctx, "cflags", utils.Join(cflagsList))
	ctx.Variable(pctx, "conlyflags", utils.Join(cctargetflags))
	ctx.Variable(pctx, "cxxflags", utils.Join(cxxtargetflags))

	objectFiles := []string{}
	nonCompiledDeps := []string{}

	for _, source := range srcs {
		var rule blueprint.Rule
		args := make(map[string]string)
		switch path.Ext(source) {
		case ".s":
			args["ascompiler"] = as
			args["asflags"] = "$asflags"
			rule = asRule
		case ".S":
			// Assembly with .S suffix must be preprocessed by the C compiler
			fallthrough
		case ".c":
			args["ccompiler"] = cc
			args["cflags"] = "$cflags"
			args["conlyflags"] = "$conlyflags"
			rule = ccRule
		case ".cc":
			fallthrough
		case ".cpp":
			args["cxxcompiler"] = cxx
			args["cflags"] = "$cflags"
			args["cxxflags"] = "$cxxflags"
			rule = cxxRule
		default:
			nonCompiledDeps = append(nonCompiledDeps, getBackendPathInSourceDir(g, source))
			continue
		}

		var sourceWithoutPrefix string
		if buildDir := g.buildDir(); strings.HasPrefix(source, buildDir) {
			sourceWithoutPrefix = source[len(buildDir):]
		} else {
			sourceWithoutPrefix = source
			source = getBackendPathInSourceDir(g, source)
		}

		output := l.ObjDir() + sourceWithoutPrefix + ".o"
		ctx.Build(pctx,
			blueprint.BuildParams{
				Rule:     rule,
				Outputs:  []string{output},
				Inputs:   []string{source},
				Args:     args,
				Optional: true,
			})
		objectFiles = append(objectFiles, output)
	}

	return objectFiles, nonCompiledDeps
}

func (g *linuxGenerator) strictLibraryStaticActions(m *strictLibrary, ctx blueprint.ModuleContext, objectFiles []string) {
	m.Static.outputdir = m.ObjDir()
	m.Static.outs = []string{filepath.Join(m.Static.outputDir(), m.Name()+".a")}

	tc := g.getToolchain(m.Properties.TargetType)
	arBinary, _ := tc.getArchiver()

	depfiles := []string{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(m) == staticDepTag
		},
		func(m blueprint.Module) {
			gen, _ := m.(*strictLibrary)
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

func (g *linuxGenerator) strictLibrarySharedActions(m *strictLibrary, ctx blueprint.ModuleContext, objectFiles []string) {
	m.Shared.outputdir = g.sharedLibsDir(m.Properties.TargetType)
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

	tc := g.getToolchain(m.Properties.TargetType)
	linker := tc.getLinker().getTool()
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

func (g *linuxGenerator) strictLibraryActions(m *strictLibrary, ctx blueprint.ModuleContext) {
	objectFiles, _ := m.CompileObjs(ctx)
	g.strictLibraryStaticActions(m, ctx, objectFiles)
	// TODO: Stub the shared lib implementation and break it off of this patch.
	// g.strictLibrarySharedActions(m, ctx, objectFiles)
}

func proxyCflags(m *strictLibrary) []string {
	Cflags := m.Properties.Copts
	for _, def := range m.Properties.Local_defines {
		Cflags = append(Cflags, "-D"+def)
	}
	for _, def := range m.Properties.Defines {
		Cflags = append(Cflags, "-D"+def)
	}
	return Cflags
}

func (g *androidBpGenerator) strictLibraryActions(m *strictLibrary, ctx blueprint.ModuleContext) {
	// TODO: Move this to it's own file

	// TODO: Handle shared library versions too
	var proxyStaticLib staticLibrary
	proxyStaticLib.SimpleName.Properties.Name = m.SimpleName.Properties.Name
	proxyStaticLib.Properties.EnableableProps.Required = true
	proxyStaticLib.Properties.Srcs = m.Properties.Srcs
	proxyStaticLib.Properties.Cflags = proxyCflags(m)
	proxyStaticLib.Properties.Host_supported = m.Properties.Host_supported
	proxyStaticLib.Properties.Target_supported = m.Properties.Target_supported
	// TODO: generate target for all supported target types
	proxyStaticLib.Properties.TargetType = tgtTypeHost
	g.staticActions(&proxyStaticLib, ctx)
	// TODO: Static lib dependency

}