/*
 * Copyright 2020-2023 Arm Limited.
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
	"github.com/google/blueprint/proptools"

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
// Note that this function returns a list of names instead of a single name.
// This is due to resource module that can generate multiple Blueprint modules.
// All other bob modules only return one name.
func bpModuleNamesForDep(mctx blueprint.BaseModuleContext, name string) []string {
	var dep blueprint.Module

	mctx.VisitDirectDeps(func(m blueprint.Module) {
		if m.Name() == name {
			dep = m
		} else if l, ok := getLibrary(m); ok {
			// Shared libraries may already have their shortname as name
			if l.shortName() == name {
				dep = m
			}
		}

	})

	if dep == nil {
		utils.Die("%s has no dependency '%s'", mctx.ModuleName(), name)
	}

	if r, ok := dep.(*resource); ok {
		var modNames []string

		r.Properties.GetSrcs(mctx).ForEach(
			func(fp filePath) bool {
				modNames = append(modNames, r.getAndroidbpResourceName(fp.localPath()))
				return true
			})

		if len(modNames) == 0 {
			utils.Die("bob_resource %s has empty srcs", name)
		}
		return modNames
	}

	if l, ok := getLibrary(dep); ok {
		return []string{l.shortName()}
	}

	// Most cases should match the getLibrary() check above, but generated libraries,
	// etc, do not, and they also do not require using shortName() (because of not
	// being target-specific), so just use the original build.bp name.
	return []string{dep.Name()}
}

func bpModuleNamesForDeps(mctx blueprint.BaseModuleContext, nameLists ...[]string) []string {
	ccModules := []string{}
	for _, nameList := range nameLists {
		for _, name := range nameList {
			ccModules = append(ccModules, bpModuleNamesForDep(mctx, name)...)
		}
	}
	return ccModules
}

func (m *ModuleLibrary) getGeneratedSourceModules(mctx blueprint.BaseModuleContext) (srcs []string) {
	mctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return mctx.OtherModuleDependencyTag(dep) == generatedSourceTag
		},
		func(dep blueprint.Module) {
			switch dep.(type) {
			case *ModuleGenerateSource:
			case *ModuleTransformSource:
			case *ModuleGenrule:
			default:
				panic(fmt.Errorf("Dependency %s of %s is not a generated source",
					dep.Name(), m.Name()))
			}

			srcs = append(srcs, dep.Name())
		})
	return
}

func (m *ModuleLibrary) getGeneratedHeaderModules(mctx blueprint.BaseModuleContext) (headers, exportHeaders []string) {
	mctx.VisitDirectDeps(
		func(dep blueprint.Module) {
			switch mctx.OtherModuleDependencyTag(dep) {
			case generatedHeaderTag:
				headers = append(headers, dep.Name())
			case exportGeneratedHeaderTag:
				exportHeaders = append(exportHeaders, dep.Name())
			default:
				return
			}

			switch dep.(type) {
			case *ModuleGenerateSource:
			case *ModuleTransformSource:
			case *ModuleGenrule:
			default:
				panic(fmt.Errorf("Dependency %s of %s is not a generated source",
					dep.Name(), m.Name()))
			}
		})
	return
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

func addMTEProps(m bpwriter.Module, props AndroidMTEProps) {
	memtagHeap := proptools.Bool(props.Mte.Memtag_heap)
	diagMemtagHeap := proptools.Bool(props.Mte.Diag_memtag_heap)

	if !memtagHeap {
		return
	}

	g := m.NewGroup("sanitize")
	g.AddBool("memtag_heap", true)

	if diagMemtagHeap {
		diag := g.NewGroup("diag")
		diag.AddBool("memtag_heap", true)
	}
}

func addHWASANProps(m bpwriter.Module, props Build) {
	memtagHeap := proptools.Bool(props.AndroidMTEProps.Mte.Memtag_heap)
	if memtagHeap {
		return
	}
	if proptools.Bool(props.Hwasan_enabled) {
		g := m.NewGroup("sanitize")
		g.AddBool("hwaddress", true)
	}
}

func addRequiredModules(mod bpwriter.Module, m ModuleLibrary, mctx blueprint.ModuleContext) {
	if _, _, ok := getSoongInstallPath(m.getInstallableProps()); ok {
		requiredModuleNames := m.getInstallDepPhonyNames(mctx)
		mod.AddStringList("required", bpModuleNamesForDeps(mctx, requiredModuleNames))
	}
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

func (g *androidBpGenerator) getVersionScript(m *ModuleLibrary, ctx blueprint.ModuleContext) *string {
	if m.Properties.VersionScriptModule != nil {
		value := ":" + *m.Properties.VersionScriptModule
		return &value
	}
	return m.Properties.Build.Version_script
}

func addCcLibraryProps(mod bpwriter.Module, m ModuleLibrary, mctx blueprint.ModuleContext) {
	if len(m.Properties.Export_include_dirs) > 0 {
		utils.Die("Module %s exports non-local include dirs %v - this is not supported",
			mctx.ModuleName(), m.Properties.Export_include_dirs)
	}

	if len(m.Properties.Export_system_include_dirs) > 0 {
		utils.Die("Module %s exports non-local system include dirs %v - this is not supported",
			mctx.ModuleName(), m.Properties.Export_system_include_dirs)
	}

	// Soong deals with exported include directories between library
	// modules, but it doesn't export cflags.
	_, _, _, _, exported_cflags := m.GetExportedVariables(mctx)

	cflags := utils.NewStringSlice(m.Properties.Cflags, m.Properties.Export_cflags, exported_cflags)

	sharedLibs := bpModuleNamesForDeps(mctx, m.Properties.Shared_libs)
	staticLibs := bpModuleNamesForDeps(mctx, m.Properties.ResolvedStaticLibs)
	// Exported header libraries must be mentioned in both header_libs
	// *and* export_header_lib_headers - i.e., we can't export a header
	// library which isn't actually being used.
	headerLibs := bpModuleNamesForDeps(mctx, m.Properties.Header_libs, m.Properties.Export_header_libs)

	reexportShared := []string{}
	reexportStatic := []string{}
	reexportHeaders := bpModuleNamesForDeps(mctx, m.Properties.Export_header_libs)
	for _, lib := range bpModuleNamesForDeps(mctx, m.Properties.Reexport_libs) {
		if utils.Contains(sharedLibs, lib) {
			reexportShared = append(reexportShared, lib)
		} else if utils.Contains(staticLibs, lib) {
			reexportStatic = append(reexportStatic, lib)
		} else if utils.Contains(headerLibs, lib) {
			reexportHeaders = append(reexportHeaders, lib)
		}
	}

	if m.shortName() != m.outputName() {
		mod.AddString("stem", m.outputName())
	}

	srcs := []string{}
	m.Properties.GetSrcs(mctx).ForEachIf(
		func(fp filePath) bool {
			// On Android, generated sources are passed to the modules via
			// `generated_sources` so they are omitted here.
			_, isGen := fp.(generatedFilePath)
			return utils.IsCompilableSource(fp.localPath()) && !isGen
		},
		func(fp filePath) bool {
			srcs = append(srcs, fp.localPath())
			return true
		})

	mod.AddStringList("srcs", srcs)

	generated_srcs := m.getGeneratedSourceModules(mctx)
	mod.AddStringList("generated_sources", generated_srcs)

	genHeaderModules, exportGenHeaderModules := m.getGeneratedHeaderModules(mctx)
	mod.AddStringList("generated_headers", append(genHeaderModules, exportGenHeaderModules...))
	mod.AddStringList("export_generated_headers", exportGenHeaderModules)
	mod.AddStringList("exclude_srcs", m.Properties.Exclude_srcs)
	err := addCFlags(mod, cflags, m.Properties.Conlyflags, m.Properties.Cxxflags)
	if err != nil {
		utils.Die("Module %s: %s", mctx.ModuleName(), err.Error())
	}
	mod.AddStringList("include_dirs", m.Properties.Include_dirs)

	/* Despite the documentation Export_local_system_include_dirs is not added to local includes for the current module, and only
	propagated to downstream deps. To remedy this, we add those paths to local includes also. */
	localIncludeDirs := append(m.Properties.Local_include_dirs, m.Properties.Export_local_system_include_dirs...)
	mod.AddStringList("local_include_dirs", localIncludeDirs)
	mod.AddStringList("shared_libs", bpModuleNamesForDeps(mctx, m.Properties.Shared_libs))
	mod.AddStringList("static_libs", staticLibs)
	mod.AddStringList("whole_static_libs", bpModuleNamesForDeps(mctx, m.Properties.Whole_static_libs))
	mod.AddStringList("header_libs", headerLibs)
	mod.AddStringList("export_shared_lib_headers", reexportShared)
	mod.AddStringList("export_static_lib_headers", reexportStatic)
	mod.AddStringList("export_header_lib_headers", reexportHeaders)
	mod.AddStringList("ldflags", utils.Filter(ccflags.AndroidLinkFlags, m.Properties.Ldflags))

	_, installRel, ok := getSoongInstallPath(m.getInstallableProps())
	if ok && installRel != "" {
		mod.AddString("relative_install_path", installRel)
	}

	addProvenanceProps(mod, m.Properties.Build.AndroidProps)
	addPGOProps(mod, m.Properties.Build.AndroidPGOProps)
	addRequiredModules(mod, m, mctx)

	if m.Properties.Post_install_cmd != nil ||
		m.Properties.Post_install_args != nil ||
		m.Properties.Post_install_tool != nil {
		utils.Die("Module %s has post install actions - this is not supported on Android.bp",
			mctx.ModuleName())
	}
}

func addBinaryProps(mod bpwriter.Module, m ModuleBinary, mctx blueprint.ModuleContext) {
	// Handle installation
	if _, installRel, ok := getSoongInstallPath(m.getInstallableProps()); ok {
		// Only setup multilib for target modules.
		// We support multilib target binaries to allow creation of test
		// binaries in both modes.
		// We disable multilib if this module depends on generated libraries
		// (which can't support multilib).
		if m.Properties.TargetType == tgtTypeTarget && !linksToGeneratedLibrary(mctx) {
			mod.AddString("compile_multilib", "both")

			// For executables we need to be clear about where to
			// install both 32 and 64 bit versions of the
			// binaries.
			g := mod.NewGroup("multilib")
			g.NewGroup("lib32").AddString("relative_install_path", installRel)
			g.NewGroup("lib64").AddString("relative_install_path", installRel+"64")
		}
	}

	addMTEProps(mod, m.Properties.Build.AndroidMTEProps)
	addHWASANProps(mod, m.Properties.Build)
}

func addStaticOrSharedLibraryProps(mod bpwriter.Module, m ModuleLibrary, mctx blueprint.ModuleContext) {
	// Soong's `export_include_dirs` field is relative to the module
	// dir. The Android.bp backend writes the file into the project
	// root, so we can use the Export_local_include_dirs and its system counter part
	// property unchanged.
	mod.AddStringList("export_include_dirs", m.Properties.Export_local_include_dirs)
	mod.AddStringList("export_system_include_dirs ", m.Properties.Export_local_system_include_dirs)

	// Only setup multilib for target modules.
	// This part handles the target libraries.
	// We disable multilib if this module depends on generated libraries
	// (which can't support multilib).
	if m.Properties.TargetType == tgtTypeTarget && !linksToGeneratedLibrary(mctx) {
		mod.AddString("compile_multilib", "both")
	}
	addHWASANProps(mod, m.Properties.Build)
}

func addStripProp(m bpwriter.Module) {
	g := m.NewGroup("strip")
	g.AddBool("all", true)
}

func (g *androidBpGenerator) binaryActions(m *ModuleBinary, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	// Calculate and record outputs
	m.outs = []string{m.outputName()}

	installBase, _, _ := getSoongInstallPath(m.getInstallableProps())

	var modType string
	useCcTest := false
	if installBase == "tests" {
		useCcTest = true
		switch m.Properties.TargetType {
		case tgtTypeHost:
			modType = "cc_test_host"
		case tgtTypeTarget:
			modType = "cc_test"
		}
	} else {
		if installBase != "" && installBase != "bin" {
			panic(fmt.Errorf("Unknown base install location for %s (%s)",
				m.Name(), installBase))
		}

		switch m.Properties.TargetType {
		case tgtTypeHost:
			modType = "cc_binary_host"
		case tgtTypeTarget:
			modType = "cc_binary"
		}
	}

	mod, err := AndroidBpFile().NewModule(modType, m.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(mod, m.ModuleLibrary, mctx)
	addBinaryProps(mod, *m, mctx)
	if m.strip() {
		addStripProp(mod)
	}
	if useCcTest {
		// Avoid using cc_test default setup
		mod.AddBool("no_named_install_directory", true)
		mod.AddBool("include_build_directory", false)
		mod.AddBool("auto_gen_config", false)
		mod.AddBool("gtest", false)
	}

	versionScript := g.getVersionScript(&m.ModuleLibrary, mctx)
	if versionScript != nil {
		mod.AddString("version_script", *versionScript)
	}
}

func (g *androidBpGenerator) sharedActions(m *ModuleSharedLibrary, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	// Calculate and record outputs
	m.outs = []string{m.outputName() + m.fileNameExtension}

	var modType string
	switch m.Properties.TargetType {
	case tgtTypeHost:
		modType = "cc_library_host_shared"
	case tgtTypeTarget:
		modType = "cc_library_shared"
	}

	installBase, _, _ := getSoongInstallPath(m.getInstallableProps())
	if installBase != "" && installBase != "lib" {
		panic(fmt.Errorf("Unknown base install location for %s (%s)",
			m.Name(), installBase))
	}

	mod, err := AndroidBpFile().NewModule(modType, m.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(mod, m.ModuleLibrary, mctx)
	addStaticOrSharedLibraryProps(mod, m.ModuleLibrary, mctx)
	if m.strip() {
		addStripProp(mod)
	}

	versionScript := g.getVersionScript(&m.ModuleLibrary, mctx)
	if versionScript != nil {
		mod.AddString("version_script", *versionScript)
	}
}

func (g *androidBpGenerator) staticActions(m *ModuleStaticLibrary, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	// Calculate and record outputs
	m.outs = []string{m.outputName()}

	var modType string
	switch m.Properties.TargetType {
	case tgtTypeHost:
		modType = "cc_library_host_static"
	case tgtTypeTarget:
		modType = "cc_library_static"
	}

	mod, err := AndroidBpFile().NewModule(modType, m.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(mod, m.ModuleLibrary, mctx)
	addStaticOrSharedLibraryProps(mod, m.ModuleLibrary, mctx)
}
