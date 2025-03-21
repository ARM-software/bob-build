package core

// Support for building libraries and binaries via soong's cc_library
// modules.

import (
	"fmt"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
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
func bpModuleNamesForDep(ctx blueprint.BaseModuleContext, name string) []string {
	var dep blueprint.Module

	ctx.VisitDirectDeps(func(m blueprint.Module) {
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
		utils.Die("%s has no dependency '%s'", ctx.ModuleName(), name)
	}

	if r, ok := dep.(*ModuleResource); ok {
		var modNames []string

		r.Properties.GetFiles(ctx).ForEach(
			func(fp file.Path) bool {
				modNames = append(modNames, r.getAndroidbpResourceName(fp.UnScopedPath()))
				return true
			})

		if len(modNames) == 0 {
			utils.Die("bob_resource %s has empty srcs", name)
		}
		return modNames
	}

	if l, ok := getLibrary(dep); ok {
		return []string{l.shortName()}
	} else if l, ok := dep.(*ModuleStrictLibrary); ok {
		return []string{l.shortName()}
	}

	// Most cases should match the getLibrary() check above, but generated libraries,
	// etc, do not, and they also do not require using shortName() (because of not
	// being target-specific), so just use the original build.bp name.
	return []string{dep.Name()}
}

func bpModuleNamesForDeps(ctx blueprint.BaseModuleContext, nameLists ...[]string) []string {
	ccModules := []string{}
	for _, nameList := range nameLists {
		for _, name := range nameList {
			ccModules = append(ccModules, bpModuleNamesForDep(ctx, name)...)
		}
	}
	return ccModules
}

func (m *ModuleLibrary) getGeneratedSourceModules(ctx blueprint.BaseModuleContext) (srcs []string) {
	ctx.VisitDirectDepsIf(
		func(dep blueprint.Module) bool {
			return ctx.OtherModuleDependencyTag(dep) == tag.GeneratedSourcesTag
		},
		func(dep blueprint.Module) {
			switch dep.(type) {
			case *ModuleGenerateSource:
			case *ModuleTransformSource:
			case *ModuleGenrule:
			case *ModuleGensrcs:
			default:
				panic(fmt.Errorf("Dependency %s of %s is not a generated source",
					dep.Name(), m.Name()))
			}

			srcs = append(srcs, dep.Name())
		})
	return
}

func (m *ModuleLibrary) getGeneratedHeaderModules(ctx blueprint.BaseModuleContext) (headers, exportHeaders []string) {
	ctx.VisitDirectDeps(
		func(dep blueprint.Module) {
			switch ctx.OtherModuleDependencyTag(dep) {
			case tag.GeneratedHeadersTag:
				headers = append(headers, dep.Name())
			case tag.ExportGeneratedHeadersTag:
				exportHeaders = append(exportHeaders, dep.Name())
			default:
				return
			}

			switch dep.(type) {
			case *ModuleGenerateSource:
			case *ModuleTransformSource:
			case *ModuleGenrule:
			case *ModuleGensrcs:
			default:
				panic(fmt.Errorf("Dependency %s of %s is not a generated source",
					dep.Name(), m.Name()))
			}
		})
	return
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

func addHWASANProps(m bpwriter.Module) {
	g := m.NewGroup("sanitize")
	g.AddBool("hwaddress", true)
}

func addRequiredModules(mod bpwriter.Module, m ModuleLibrary, ctx blueprint.ModuleContext) {
	if _, _, ok := getSoongInstallPath(m.getInstallableProps()); ok {
		requiredModuleNames := m.getInstallDepPhonyNames(ctx)
		mod.AddStringList("required", bpModuleNamesForDeps(ctx, requiredModuleNames))
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

func addCcLibraryProps(mod bpwriter.Module, m ModuleLibrary, ctx blueprint.ModuleContext) {
	if len(m.Properties.Export_include_dirs) > 0 {
		utils.Die("Module %s exports non-local include dirs %v - this is not supported",
			ctx.ModuleName(), m.Properties.Export_include_dirs)
	}

	if len(m.Properties.Export_system_include_dirs) > 0 {
		utils.Die("Module %s exports non-local system include dirs %v - this is not supported",
			ctx.ModuleName(), m.Properties.Export_system_include_dirs)
	}

	cflags := utils.NewStringSlice(m.Properties.Cflags, m.Properties.Export_cflags)

	m.FlagsInTransitive(ctx).Filtered(
		func(f flag.Flag) bool {
			// Soong deals with exported include directories between library
			// modules, but it doesn't export cflags.
			return f.MatchesType(flag.TypeExported) &&
				f.MatchesType(flag.TypeCC) &&
				!f.MatchesType(flag.TypeInclude)
		},
	).ForEach(
		func(f flag.Flag) {
			cflags = append(cflags, f.ToString())
		},
	)

	sharedLibs := bpModuleNamesForDeps(ctx, m.Properties.Shared_libs)
	staticLibs := bpModuleNamesForDeps(ctx, m.Properties.ResolvedStaticLibs)
	// Exported header libraries must be mentioned in both header_libs
	// *and* export_header_lib_headers - i.e., we can't export a header
	// library which isn't actually being used.
	headerLibs := bpModuleNamesForDeps(ctx, m.Properties.Header_libs, m.Properties.Export_header_libs)

	reexportShared := []string{}
	reexportStatic := []string{}
	reexportHeaders := bpModuleNamesForDeps(ctx, m.Properties.Export_header_libs)
	for _, lib := range bpModuleNamesForDeps(ctx, m.Properties.Reexport_libs) {
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
	m.Properties.GetFiles(ctx).ForEachIf(
		func(fp file.Path) bool {
			// On Android, generated sources are passed to the modules via
			// `generated_sources` so they are omitted here.
			return fp.IsType(file.TypeCompilable) && fp.IsNotType(file.TypeGenerated)
		},
		func(fp file.Path) bool {
			srcs = append(srcs, fp.UnScopedPath())
			return true
		})

	mod.AddStringList("srcs", srcs)

	generated_srcs := m.getGeneratedSourceModules(ctx)
	mod.AddStringList("generated_sources", generated_srcs)

	genHeaderModules, exportGenHeaderModules := m.getGeneratedHeaderModules(ctx)
	mod.AddStringList("generated_headers", append(genHeaderModules, exportGenHeaderModules...))
	mod.AddStringList("export_generated_headers", exportGenHeaderModules)
	mod.AddStringList("exclude_srcs", m.Properties.Exclude_srcs)
	err := addCFlags(mod, cflags, m.Properties.Conlyflags, m.Properties.Cxxflags)
	if err != nil {
		utils.Die("Module %s: %s", ctx.ModuleName(), err.Error())
	}
	mod.AddStringList("include_dirs", m.Properties.Include_dirs)

	/* Despite the documentation Export_local_system_include_dirs is not added to local includes for the current module, and only
	propagated to downstream deps. To remedy this, we add those paths to local includes also. */
	localIncludeDirs := append(m.Properties.Local_include_dirs, m.Properties.Export_local_system_include_dirs...)
	mod.AddStringList("local_include_dirs", localIncludeDirs)
	mod.AddStringList("shared_libs", bpModuleNamesForDeps(ctx, m.Properties.Shared_libs))
	mod.AddStringList("static_libs", staticLibs)
	mod.AddStringList("whole_static_libs", bpModuleNamesForDeps(ctx, m.Properties.Whole_static_libs))
	mod.AddStringList("header_libs", headerLibs)
	mod.AddStringList("export_shared_lib_headers", reexportShared)
	mod.AddStringList("export_static_lib_headers", reexportStatic)
	mod.AddStringList("export_header_lib_headers", reexportHeaders)
	mod.AddStringList("ldflags", utils.Filter(ccflags.AndroidLinkFlags, m.Properties.Ldflags))

	_, installRel, ok := getSoongInstallPath(m.getInstallableProps())
	if ok && installRel != "" {
		mod.AddString("relative_install_path", installRel)
	}

	addProvenanceProps(ctx, mod, &m)
	addRequiredModules(mod, m, ctx)

	if m.Properties.Post_install_cmd != nil ||
		m.Properties.Post_install_args != nil ||
		m.Properties.Post_install_tool != nil {
		utils.Die("Module %s has post install actions - this is not supported on Android.bp",
			ctx.ModuleName())
	}
}

func addBinaryProps(mod bpwriter.Module, m *ModuleBinary, ctx blueprint.ModuleContext, g *androidBpGenerator) {
	// Handle installation
	if _, installRel, ok := getSoongInstallPath(m.getInstallableProps()); ok {
		// Only setup multilib for target modules.
		// We support multilib target binaries to allow creation of test
		// binaries in both modes.
		// We disable multilib if this module depends on generated libraries
		// (which can't support multilib).
		// We disable multilib if the target only supports 64bit
		if m.Properties.TargetType == toolchain.TgtTypeTarget &&
			!linksToGeneratedLibrary(ctx) &&
			!backend.Get().GetToolchain(toolchain.TgtTypeTarget).Is64BitOnly() {
			mod.AddString("compile_multilib", "both")

			// For executables we need to be clear about where to
			// install both 32 and 64 bit versions of the
			// binaries.
			g := mod.NewGroup("multilib")
			g.NewGroup("lib32").AddString("relative_install_path", installRel)
			g.NewGroup("lib64").AddString("relative_install_path", installRel+"64")
		}
	}

	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		addMTEProps(mod, bc.GetMteProps(ctx))
		if !proptools.Bool(bc.GetMteProps(ctx).Mte.Diag_memtag_heap) && bc.IsHwAsanEnabled() {
			addHWASANProps(mod)
		}
	}
}

func addStaticOrSharedLibraryProps(mod bpwriter.Module, m ModuleLibrary, ctx blueprint.ModuleContext) {
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
	if m.Properties.TargetType == toolchain.TgtTypeTarget &&
		!linksToGeneratedLibrary(ctx) &&
		!backend.Get().GetToolchain(toolchain.TgtTypeTarget).Is64BitOnly() {
		mod.AddString("compile_multilib", "both")
	}
}

func addStripProp(m bpwriter.Module) {
	g := m.NewGroup("strip")
	g.AddBool("all", true)
}

func addCompilableProps(mod bpwriter.Module, m Compilable, ctx blueprint.ModuleContext) {
	// TODO: move this check to the module itself
	// if len(m.Properties.Export_include_dirs) > 0 {
	// 	utils.Die("Module %s exports non-local include dirs %v - this is not supported",
	// 		ctx.ModuleName(), m.Properties.Export_include_dirs)
	// }

	// if len(m.Properties.Export_system_include_dirs) > 0 {
	// 	utils.Die("Module %s exports non-local system include dirs %v - this is not supported",
	// 		ctx.ModuleName(), m.Properties.Export_system_include_dirs)
	// }

	// Android properties
	var cflags, conlyFlags, cxxFlags, ldflags, asflags,
		srcs,
		generated_sources,
		generated_headers,
		export_generated_headers,
		include_dirs,
		local_include_dirs,
		export_include_dirs,
		export_system_include_dirs,
		static_libs,
		shared_libs,
		whole_static_libs []string

	m.GetFiles(ctx).ForEachIf(
		func(fp file.Path) bool {
			// On Android, generated sources are passed to the modules via
			// `generated_sources` so they are omitted here.
			return fp.IsType(file.TypeCompilable) && fp.IsNotType(file.TypeGenerated)
		},
		func(fp file.Path) bool {
			srcs = append(srcs, fp.UnScopedPath())
			return true
		})

	m.FlagsInTransitive(ctx).ForEachIf(
		func(f flag.Flag) bool {
			// Includes are not part of cflags on AOSP, and the exporting of them is done by Soong itself,
			// hence exclude processing of includes here and handle it separately.
			return !f.MatchesType(flag.TypeInclude)
		},
		func(f flag.Flag) {
			switch {
			case (f.Type() & flag.TypeCompilable) == flag.TypeC: //c exclusive flags
				conlyFlags = append(conlyFlags, f.ToString())
			case f.MatchesType(flag.TypeCC | flag.TypeInclude):
				cflags = append(cflags, f.ToString())
			case f.MatchesType(flag.TypeCpp):
				cxxFlags = append(cxxFlags, f.ToString())
			case f.MatchesType(flag.TypeLinker):
				ldflags = append(ldflags, f.ToString())
			case f.MatchesType(flag.TypeTransitiveLinker):
				ldflags = append(ldflags, f.ToString())
			case f.MatchesType(flag.TypeAsm):
				asflags = append(asflags, f.ToString())
			}
		},
	)

	// Process includes for current module only.
	m.FlagsIn().ForEachIf(
		func(f flag.Flag) bool {
			return f.MatchesType(flag.TypeInclude)
		},
		func(f flag.Flag) {
			if f.MatchesType(flag.TypeIncludeLocal) {
				local_include_dirs = append(local_include_dirs, f.Raw())
			} else {
				include_dirs = append(include_dirs, f.Raw())
			}
		},
	)

	// Check if any includes are being exported
	m.FlagsOut().ForEachIf(
		func(f flag.Flag) bool {
			return f.MatchesType(flag.TypeInclude) &&
				f.MatchesType(flag.TypeExported) &&
				f.MatchesType(flag.TypeIncludeLocal)
		},
		func(f flag.Flag) {
			if f.MatchesType(flag.TypeIncludeSystem) {
				export_system_include_dirs = append(export_system_include_dirs, f.Raw())
			} else {
				export_include_dirs = append(export_include_dirs, f.Raw())
			}
		},
	)

	ctx.VisitDirectDeps(
		func(dep blueprint.Module) {
			switch ctx.OtherModuleDependencyTag(dep) {
			case tag.SharedTag:
				shared_libs = append(shared_libs, bpModuleNamesForDep(ctx, dep.Name())...)
			case tag.StaticTag:
				static_libs = append(static_libs, bpModuleNamesForDep(ctx, dep.Name())...)
			case tag.WholeStaticTag:
				whole_static_libs = append(whole_static_libs, bpModuleNamesForDep(ctx, dep.Name())...)
			case tag.GeneratedSourcesTag:
				generated_sources = append(generated_sources, bpModuleNamesForDep(ctx, dep.Name())...)
			case tag.GeneratedHeadersTag:
				generated_headers = append(generated_headers, bpModuleNamesForDep(ctx, dep.Name())...)
			case tag.ExportGeneratedHeadersTag:
				export_generated_headers = append(export_generated_headers, dep.Name())
			}
		})

	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		addMTEProps(mod, bc.GetMteProps(ctx))
	}

	if std := ccflags.GetCompilerStandard(cflags, conlyFlags); std != "" {
		mod.AddString("c_std", std)
	}

	if std := ccflags.GetCompilerStandard(cflags, cxxFlags); std != "" {
		mod.AddString("cpp_std", std)
	}

	if armMode, err := ccflags.GetArmMode(cflags, conlyFlags, cxxFlags); err == nil && armMode != "" {
		mod.AddString("instruction_set", armMode)
	}

	if m.shortName() != m.outputName() {
		mod.AddString("stem", m.outputName())
	}

	mod.AddStringList("srcs", srcs)
	mod.AddStringList("asflags", utils.Filter(ccflags.AndroidCompileFlags, asflags))
	mod.AddStringList("cflags", utils.Filter(ccflags.AndroidCompileFlags, cflags))
	mod.AddStringList("conlyflags", utils.Filter(ccflags.AndroidCompileFlags, conlyFlags))
	mod.AddStringList("cppflags", utils.Filter(ccflags.AndroidCompileFlags, cxxFlags))
	mod.AddStringList("ldflags", utils.Filter(ccflags.AndroidLinkFlags, ldflags))
	mod.AddStringList("generated_sources", generated_sources)
	mod.AddStringList("generated_headers", generated_headers)
	mod.AddStringList("export_generated_headers", export_generated_headers)

	mod.AddStringList("include_dirs", include_dirs)
	mod.AddStringList("local_include_dirs", local_include_dirs)

	// only `cc_library` contains those properties
	_, ok1 := ctx.Module().(*ModuleStrictBinary)
	_, ok2 := ctx.Module().(*ModuleTest)
	if !ok1 && !ok2 {
		mod.AddStringList("export_include_dirs", export_include_dirs)
		mod.AddStringList("export_system_include_dirs ", export_system_include_dirs)
	}

	mod.AddStringList("shared_libs", shared_libs)
	mod.AddStringList("static_libs", static_libs)
	mod.AddStringList("whole_static_libs", whole_static_libs)
}

func (g *androidBpGenerator) binaryActions(m *ModuleBinary, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	installBase, _, _ := getSoongInstallPath(m.getInstallableProps())

	var modType string
	useCcTest := false
	if installBase == "tests" {
		useCcTest = true
		switch m.Properties.TargetType {
		case toolchain.TgtTypeHost:
			modType = "cc_test_host"
		case toolchain.TgtTypeTarget:
			modType = "cc_test"
		}
	} else {
		if installBase != "" && installBase != "bin" {
			panic(fmt.Errorf("Unknown base install location for %s (%s)",
				m.Name(), installBase))
		}

		switch m.Properties.TargetType {
		case toolchain.TgtTypeHost:
			modType = "cc_binary_host"
		case toolchain.TgtTypeTarget:
			modType = "cc_binary"
		}
	}

	mod, err := AndroidBpFile().NewModule(modType, m.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(mod, m.ModuleLibrary, ctx)
	addBinaryProps(mod, m, ctx, g)
	bc := GetModuleBackendConfiguration(ctx, m)
	if bc.strip() {
		addStripProp(mod)
	}

	if useCcTest {
		// Avoid using cc_test default setup
		mod.AddBool("no_named_install_directory", true)
		mod.AddBool("include_build_directory", false)
		mod.AddBool("auto_gen_config", false)
		mod.AddBool("gtest", false)
	}

	versionScript := g.getVersionScript(&m.ModuleLibrary, ctx)
	if versionScript != nil {
		mod.AddString("version_script", *versionScript)
	}
}

func (g *androidBpGenerator) sharedActions(m *ModuleSharedLibrary, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	var modType string
	switch m.Properties.TargetType {
	case toolchain.TgtTypeHost:
		modType = "cc_library_host_shared"
	case toolchain.TgtTypeTarget:
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

	addCcLibraryProps(mod, m.ModuleLibrary, ctx)
	addStaticOrSharedLibraryProps(mod, m.ModuleLibrary, ctx)

	bc := GetModuleBackendConfiguration(ctx, m)
	if bc.strip() {
		addStripProp(mod)
	}

	if !proptools.Bool(bc.GetMteProps(ctx).Mte.Diag_memtag_heap) && bc.IsHwAsanEnabled() {
		addHWASANProps(mod)
	}

	versionScript := g.getVersionScript(&m.ModuleLibrary, ctx)
	if versionScript != nil {
		mod.AddString("version_script", *versionScript)
	}
}

func (g *androidBpGenerator) staticActions(m *ModuleStaticLibrary, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	var modType string
	switch m.Properties.TargetType {
	case toolchain.TgtTypeHost:
		modType = "cc_library_host_static"
	case toolchain.TgtTypeTarget:
		modType = "cc_library_static"
	}

	mod, err := AndroidBpFile().NewModule(modType, m.shortName())
	if err != nil {
		panic(err.Error())
	}

	addCcLibraryProps(mod, m.ModuleLibrary, ctx)
	addStaticOrSharedLibraryProps(mod, m.ModuleLibrary, ctx)

	bc := GetModuleBackendConfiguration(ctx, m)
	if bc != nil {
		if !proptools.Bool(bc.GetMteProps(ctx).Mte.Diag_memtag_heap) && bc.IsHwAsanEnabled() {
			addHWASANProps(mod)
		}
	}
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
	if !enabledAndRequired(m) {
		return
	}

	mod, err := AndroidBpFile().NewModule("cc_library", m.shortName())
	if err != nil {
		panic(err.Error())
	}

	bc := GetModuleBackendConfiguration(ctx, m)

	switch m.Properties.TargetType {
	case toolchain.TgtTypeHost:
		mod.AddBool("host_supported", true)
		mod.AddBool("device_supported", false)

	case toolchain.TgtTypeTarget:
		mod.AddBool("host_supported", false)
		mod.AddBool("device_supported", true)
	}

	addCompilableProps(mod, m, ctx)

	if bc != nil && bc.strip() {
		addStripProp(mod)
	}

	// TODO: Enable relative install path
	// _, installRel, ok := getSoongInstallPath(m.getInstallableProps())
	// if ok && installRel != "" {
	// 	mod.AddString("relative_install_path", installRel)
	// }

	addProvenanceProps(ctx, mod, m)

	if bc != nil && !proptools.Bool(bc.GetMteProps(ctx).Mte.Diag_memtag_heap) && bc.IsHwAsanEnabled() {
		addHWASANProps(mod)
	}

	// TODO: Make addRequiredModules generic and enable it if needed
	// addRequiredModules(mod, m, ctx)

	if m.Properties.TargetType == toolchain.TgtTypeTarget &&
		!linksToGeneratedLibrary(ctx) &&
		!backend.Get().GetToolchain(toolchain.TgtTypeTarget).Is64BitOnly() {
		mod.AddString("compile_multilib", "both")
	}
}

func (g *androidBpGenerator) executableTestActions(m *ModuleTest, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	var modType string

	switch m.Properties.TargetType {
	case toolchain.TgtTypeHost:
		modType = "cc_test_host"
	case toolchain.TgtTypeTarget:
		modType = "cc_test"
	}

	mod, err := AndroidBpFile().NewModule(modType, m.shortName())
	if err != nil {
		panic(err.Error())
	}

	bc := GetModuleBackendConfiguration(ctx, m)

	addCompilableProps(mod, m, ctx)

	if bc != nil && bc.strip() {
		addStripProp(mod)
	}

	// TODO: For executables we need to be clear about where to
	// install both 32 and 64 bit versions of the
	// binaries.
	// if m.Properties.TargetType == toolchain.TgtTypeTarget && !linksToGeneratedLibrary(ctx) {
	// 	mod.AddString("compile_multilib", "both")
	// }

	addProvenanceProps(ctx, mod, m)

	if bc != nil && !proptools.Bool(bc.GetMteProps(ctx).Mte.Diag_memtag_heap) && bc.IsHwAsanEnabled() {
		addHWASANProps(mod)
	}

	// Avoid using cc_test default setup
	// TODO: `relative_install_path` needed - Module install directory may only be disabled if relative_install_path is set
	// mod.AddBool("no_named_install_directory", true)
	mod.AddBool("include_build_directory", false)
	mod.AddBool("auto_gen_config", false)
	mod.AddBool("gtest", false)
}

func (g *androidBpGenerator) strictBinaryActions(m *ModuleStrictBinary, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	var modtype string
	switch m.Properties.TargetType {
	case toolchain.TgtTypeHost:
		modtype = "cc_binary_host"
	case toolchain.TgtTypeTarget:
		modtype = "cc_binary"
	}

	mod, err := AndroidBpFile().NewModule(modtype, m.shortName())
	if err != nil {
		panic(err.Error())
	}
	bc := GetModuleBackendConfiguration(ctx, m)

	addCompilableProps(mod, m, ctx)

	if bc != nil && bc.strip() {
		addStripProp(mod)
	}

	if bc != nil && !proptools.Bool(bc.GetMteProps(ctx).Mte.Diag_memtag_heap) && bc.IsHwAsanEnabled() {
		addHWASANProps(mod)
	}

	if m.Properties.TargetType == toolchain.TgtTypeTarget &&
		!linksToGeneratedLibrary(ctx) &&
		!backend.Get().GetToolchain(toolchain.TgtTypeTarget).Is64BitOnly() {
		mod.AddString("compile_multilib", "both")
	}

	addProvenanceProps(ctx, mod, m)
}
