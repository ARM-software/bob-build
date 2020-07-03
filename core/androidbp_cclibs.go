/*
 * Copyright 2020 Arm Limited.
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

// Support for building libraries and binaries via soong's cc_library
// modules.

import (
	"fmt"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/ccflags"
	"github.com/ARM-software/bob-build/internal/utils"
)

// Convert between Bob module names, and the name we will give the generated
// cc_library module. This is required when a module supports being built on
// host and target; we cannot create two modules with the same name, so
// instead, we use the `shortName()` (which may include a `__host` or
// `__target` suffix) to disambiguate, and use the `stem` property to fix up
// the output filename.
func ccModuleName(mctx blueprint.BaseModuleContext, name string) string {
	var dep blueprint.Module

	mctx.VisitDirectDeps(func(m blueprint.Module) {
		if m.Name() == name {
			dep = m
		}
	})

	if dep == nil {
		panic(fmt.Errorf("%s has no dependency '%s'", mctx.ModuleName(), name))
	}

	if l, ok := getLibrary(dep); ok {
		return l.shortName()
	}

	// Most cases should match the getLibrary() check above, but generated libraries,
	// etc, do not, and they also do not require using shortName() (because of not
	// being target-specific), so just use the original build.bp name.
	return dep.Name()
}

func ccModuleNames(mctx blueprint.BaseModuleContext, nameLists ...[]string) []string {
	ccModules := []string{}
	for _, nameList := range nameLists {
		for _, name := range nameList {
			ccModules = append(ccModules, ccModuleName(mctx, name))
		}
	}
	return ccModules
}

func (l *library) getGeneratedSourceModules(mctx blueprint.BaseModuleContext) (srcs []string) {
	mctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return mctx.OtherModuleDependencyTag(dep) == generatedSourceTag
		},
		func(dep blueprint.Module) {
			switch dep.(type) {
			case *generateSource:
			case *transformSource:
			default:
				panic(fmt.Errorf("Dependency %s of %s is not a generated source",
					dep.Name(), l.Name()))
			}

			srcs = append(srcs, dep.Name())
		})
	return
}

func (l *library) getGeneratedHeaderModules(mctx blueprint.BaseModuleContext) (headers []string) {
	mctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return mctx.OtherModuleDependencyTag(dep) == generatedHeaderTag
		},
		func(dep blueprint.Module) {
			switch dep.(type) {
			case *generateSource:
			case *transformSource:
			default:
				panic(fmt.Errorf("Dependency %s of %s is not a generated source",
					dep.Name(), l.Name()))
			}

			headers = append(headers, dep.Name())
		})
	return
}

func addProvenanceProps(m bpwriter.Module, props AndroidProps) {
	if props.Owner != "" {
		m.AddString("owner", props.Owner)
		m.AddBool("vendor", true)
		m.AddBool("proprietary", true)
		m.AddBool("soc_specific", true)
	}
}

func addPGOProps(m bpwriter.Module, props AndroidPGOProps) {
	if props.Pgo.Profile_file == nil {
		return
	}

	g := m.NewGroup("pgo")

	// `instrumentation` controls whether PGO is used for this module. This function checks
	// `profile_file` at the start; if it was set, we can infer that PGO is being used.
	g.AddBool("instrumentation", true)

	// Sampling-based PGO is not currently supported, so Soong only allows
	// this to be false. There is therefore no need to set it explicitly.
	// g.AddBool("sampling", false)
	g.AddStringList("benchmarks", props.Pgo.Benchmarks)

	g.AddString("profile_file", *props.Pgo.Profile_file)

	// If not overridden explicitly, don't set it, which will result in
	// Soong's default value of `true` being used.
	g.AddOptionalBool("enable_profile_use", props.Pgo.Enable_profile_use)

	g.AddStringList("cflags", props.Pgo.Cflags)
}

func addCFlags(m bpwriter.Module, cflags []string, conlyFlags []string, cxxFlags []string) error {
	if std := ccflags.GetCompilerStandard(cflags, conlyFlags); std != "" {
		m.AddString("c_std", std)
	}

	if std := ccflags.GetCompilerStandard(cflags, cxxFlags); std != "" {
		m.AddString("cpp_std", std)
	}

	armMode, err := ccflags.GetArmMode(cflags, conlyFlags, cxxFlags)
	if err != nil {
		return err
	}

	if armMode != "" {
		m.AddString("instruction_set", armMode)
	}

	m.AddStringList("cflags", utils.Filter(ccflags.AndroidCompileFlags, cflags))
	m.AddStringList("conlyflags", utils.Filter(ccflags.AndroidCompileFlags, conlyFlags))
	m.AddStringList("cppflags", utils.Filter(ccflags.AndroidCompileFlags, cxxFlags))
	return nil
}

func addCcLibraryProps(m bpwriter.Module, l library, mctx blueprint.ModuleContext) {
	if len(l.Properties.Export_include_dirs) > 0 {
		panic(fmt.Errorf("Module %s exports non-local include dirs %v - this is not supported",
			mctx.ModuleName(), l.Properties.Export_include_dirs))
	}

	// Soong deals with exported include directories between library
	// modules, but it doesn't export cflags.
	_, _, exported_cflags := l.GetExportedVariables(mctx)

	cflags := utils.NewStringSlice(l.Properties.Cflags, l.Properties.Export_cflags, exported_cflags)

	sharedLibs := ccModuleNames(mctx, l.Properties.Shared_libs)
	staticLibs := ccModuleNames(mctx, l.Properties.ResolvedStaticLibs)
	// Exported header libraries must be mentioned in both header_libs
	// *and* export_header_lib_headers - i.e., we can't export a header
	// library which isn't actually being used.
	headerLibs := ccModuleNames(mctx, l.Properties.Header_libs, l.Properties.Export_header_libs)

	reexportShared := []string{}
	reexportStatic := []string{}
	reexportHeaders := ccModuleNames(mctx, l.Properties.Export_header_libs)
	for _, lib := range ccModuleNames(mctx, l.Properties.Reexport_libs) {
		if utils.Contains(sharedLibs, lib) {
			reexportShared = append(reexportShared, lib)
		} else if utils.Contains(staticLibs, lib) {
			reexportStatic = append(reexportStatic, lib)
		} else if utils.Contains(headerLibs, lib) {
			reexportHeaders = append(reexportHeaders, lib)
		}
	}

	if l.shortName() != l.outputName() {
		m.AddString("stem", l.outputName())
	}
	m.AddStringList("srcs", utils.Filter(utils.IsCompilableSource, l.Properties.Srcs))
	m.AddStringList("generated_sources", l.getGeneratedSourceModules(mctx))
	genHeaderModules := l.getGeneratedHeaderModules(mctx)
	m.AddStringList("generated_headers", genHeaderModules)
	m.AddStringList("export_generated_headers", genHeaderModules)
	m.AddStringList("exclude_srcs", l.Properties.Exclude_srcs)
	err := addCFlags(m, cflags, l.Properties.Conlyflags, l.Properties.Cxxflags)
	if err != nil {
		panic(fmt.Errorf("Module %s: %s", mctx.ModuleName(), err.Error()))
	}
	m.AddStringList("include_dirs", l.Properties.Include_dirs)
	m.AddStringList("local_include_dirs", l.Properties.Local_include_dirs)
	m.AddStringList("shared_libs", ccModuleNames(mctx, l.Properties.Shared_libs))
	m.AddStringList("static_libs", staticLibs)
	m.AddStringList("whole_static_libs", ccModuleNames(mctx, l.Properties.Whole_static_libs))
	m.AddStringList("header_libs", headerLibs)
	m.AddStringList("export_shared_lib_headers", reexportShared)
	m.AddStringList("export_static_lib_headers", reexportStatic)
	m.AddStringList("export_header_lib_headers", reexportHeaders)
	m.AddStringList("ldflags", l.Properties.Ldflags)

	_, installRel, ok := getSoongInstallPath(l.getInstallableProps())
	if ok && installRel != "" {
		m.AddString("relative_install_path", installRel)
	}

	addProvenanceProps(m, l.Properties.Build.AndroidProps)
	addPGOProps(m, l.Properties.Build.AndroidPGOProps)
}

func addBinaryProps(m bpwriter.Module, l binary, mctx blueprint.ModuleContext) {
	// Handle installation
	if _, installRel, ok := getSoongInstallPath(l.getInstallableProps()); ok {
		// Only setup multilib for target modules.
		// We support multilib target binaries to allow creation of test
		// binaries in both modes.
		// We disable multilib if this module depends on generated libraries
		// (which can't support multilib).
		if l.Properties.TargetType == tgtTypeTarget && !linksToGeneratedLibrary(mctx) {
			m.AddString("compile_multilib", "both")

			// For executables we need to be clear about where to
			// install both 32 and 64 bit versions of the
			// binaries.
			g := m.NewGroup("multilib")
			g.NewGroup("lib32").AddString("relative_install_path", installRel)
			g.NewGroup("lib64").AddString("relative_install_path", installRel+"64")
		}
	}
}

func addStaticOrSharedLibraryProps(m bpwriter.Module, l library, mctx blueprint.ModuleContext) {
	// Soong's `export_include_dirs` field is relative to the module
	// dir. The Android.bp backend writes the file into the project
	// root, so we can use the Export_local_include_dirs property
	// unchanged.
	m.AddStringList("export_include_dirs", l.Properties.Export_local_include_dirs)

	// Only setup multilib for target modules.
	// This part handles the target libraries.
	// We disable multilib if this module depends on generated libraries
	// (which can't support multilib).
	if l.Properties.TargetType == tgtTypeTarget && !linksToGeneratedLibrary(mctx) {
		m.AddString("compile_multilib", "both")
	}
}

func addStripProp(m bpwriter.Module) {
	g := m.NewGroup("strip")
	g.AddBool("all", true)
}

func (g *androidBpGenerator) binaryActions(l *binary, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(l) {
		return
	}

	// Calculate and record outputs
	l.outs = []string{l.outputName()}

	var modType string
	switch l.Properties.TargetType {
	case tgtTypeHost:
		modType = "cc_binary_host"
	case tgtTypeTarget:
		modType = "cc_binary"
	}

	m, err := AndroidBpFile().NewModule(modType, l.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(m, l.library, mctx)
	addBinaryProps(m, *l, mctx)
	if l.strip() {
		addStripProp(m)
	}
}

func (g *androidBpGenerator) sharedActions(l *sharedLibrary, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(l) {
		return
	}

	// Calculate and record outputs
	l.outs = []string{l.outputName() + l.fileNameExtension}

	var modType string
	switch l.Properties.TargetType {
	case tgtTypeHost:
		modType = "cc_library_host_shared"
	case tgtTypeTarget:
		modType = "cc_library_shared"
	}

	m, err := AndroidBpFile().NewModule(modType, l.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(m, l.library, mctx)
	addStaticOrSharedLibraryProps(m, l.library, mctx)
	if l.strip() {
		addStripProp(m)
	}
}

func (g *androidBpGenerator) staticActions(l *staticLibrary, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(l) {
		return
	}

	// Calculate and record outputs
	l.outs = []string{l.outputName()}

	var modType string
	switch l.Properties.TargetType {
	case tgtTypeHost:
		modType = "cc_library_host_static"
	case tgtTypeTarget:
		modType = "cc_library_static"
	}

	m, err := AndroidBpFile().NewModule(modType, l.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(m, l.library, mctx)
	addStaticOrSharedLibraryProps(m, l.library, mctx)
}
