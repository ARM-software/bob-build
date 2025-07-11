package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

var depOutputsVarRegexp = regexp.MustCompile(`^\$\{(.+)_out\}$`)

type ExternalableProps struct {
	// Used to disable the generation of build rules. If this is set to false, no build rule will be generated.
	External *bool
}

// ModuleLibrary is a base class for modules which are generated from sets of object files
type ModuleLibrary struct {
	module.ModuleBase

	Properties struct {
		Features
		TransitiveLibraryProps
		Build
		TagableProps
		ExternalableProps

		// The list of default properties that should prepended to all configuration
		Defaults []string

		VersionScriptModule *string `blueprint:"mutated"`
	}
}

// library supports the following functionality:
type libraryInterface interface {
	aliasable
	defaultable
	enableable
	Featurable
	installable
	matchSourceInterface
	propertyEscapeInterface
	SharedLibraryExporter
	file.Consumer
	flag.Consumer
	Tagable
	BackendConfiguration
	BackendConfigurationProvider
}

// TODO: These interfaces are causing a go build issue with 'duplicate functions'
// when added to the group interface investigate why that is and fix it.
var _ splittable = (*ModuleLibrary)(nil)            // impl check
var _ targetSpecificLibrary = (*ModuleLibrary)(nil) // impl check

func (m *ModuleLibrary) defaults() []string {
	return m.Properties.Defaults
}

func (m *ModuleLibrary) defaultableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.TagableProps,
	}
}

func (m *ModuleLibrary) build() *Build {
	return &m.Properties.Build
}

func (m *ModuleLibrary) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.Build.SplittableProps,
		&m.Properties.TagableProps,
		&m.Properties.ExternalableProps,
	}
}

func (m *ModuleLibrary) targetableProperties() []interface{} {
	return []interface{}{
		&m.Properties.Build.CommonProps,
		&m.Properties.Build.BuildProps,
		&m.Properties.TagableProps,
	}
}

func (m *ModuleLibrary) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleLibrary) getTarget() toolchain.TgtType {
	return m.Properties.TargetType
}

func (m *ModuleLibrary) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

// Return the shortName of dependencies which must be installed alongside the
// library. Exclude external libraries - these will never be added via
// install_deps, but may end up in shared_libs.
func (m *ModuleLibrary) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsIf(ctx,
		func(m blueprint.Module) bool {
			depTag := ctx.OtherModuleDependencyTag(m)
			// External libraries do not have a build target so don't
			// try to add a dependency on them.
			if e, ok := m.(externableLibrary); ok && e.isExternal() {
				return false
			}
			if depTag == tag.InstallTag || depTag == tag.SharedTag {
				return true
			}
			return false
		})
}

func (m *ModuleLibrary) getEnableableProps() *EnableableProps {
	return &m.Properties.Build.EnableableProps
}

type externableLibrary interface {
	isExternal() bool
	flag.Provider
}

func (m *ModuleLibrary) isExternal() bool {
	props := &m.Properties.ExternalableProps
	if props.External != nil {
		return *props.External
	}
	return false
}

func (m *ModuleLibrary) getAliasList() []string {
	return m.Properties.getAliasList()
}

func (m *ModuleLibrary) supportedVariants() (tgts []toolchain.TgtType) {
	if m.Properties.isHostSupported() {
		tgts = append(tgts, toolchain.TgtTypeHost)
	}
	if m.Properties.isTargetSupported() {
		tgts = append(tgts, toolchain.TgtTypeTarget)
	}
	return
}

func (m *ModuleLibrary) disable() {
	f := false
	m.Properties.Enabled = &f
}

func (m *ModuleLibrary) setVariant(tgt toolchain.TgtType) {
	m.Properties.TargetType = tgt
}

func (m *ModuleLibrary) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *ModuleLibrary) getTargetSpecific(tgt toolchain.TgtType) *TargetSpecific {
	return m.Properties.getTargetSpecific(tgt)
}

func (m *ModuleLibrary) outputName() string {
	if m.Properties.Out != nil {
		return *m.Properties.Out
	}
	return m.Name()
}

func (m *ModuleLibrary) getDebugInfo() *string {
	return m.Properties.getDebugInfo()
}

func (m *ModuleLibrary) getDebugPath() *string {
	return m.Properties.getDebugPath()
}

func (m *ModuleLibrary) setDebugPath(path *string) {
	m.Properties.setDebugPath(path)
}

func (m *ModuleLibrary) stripOutputDir(g generatorBackend) string {
	return getBackendPathInBuildDir(g, string(m.Properties.TargetType), "strip")
}

func (m *ModuleLibrary) altName() string {
	return m.outputName()
}

func (m *ModuleLibrary) altShortName() string {
	if len(m.supportedVariants()) > 1 {
		return m.altName() + "__" + string(m.Properties.TargetType)
	}
	return m.altName()
}

func (m *ModuleLibrary) getEscapeProperties() []*[]string {
	return []*[]string{
		&m.Properties.Asflags,
		&m.Properties.Cflags,
		&m.Properties.Conlyflags,
		&m.Properties.Cxxflags,
		&m.Properties.Ldflags}
}

func (m *ModuleLibrary) getFlagInLut() flag.FlagParserTable {
	return flag.FlagParserTable{
		{
			PropertyName: "Cflags",
			Tag:          flag.TypeCC,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Export_cflags",
			Tag:          flag.TypeCC,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Asflags",
			Tag:          flag.TypeAsm,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Conlyflags",
			Tag:          flag.TypeC,
			Factory:      flag.FromStringOwned,
		},

		{
			PropertyName: "Cxxflags",
			Tag:          flag.TypeCpp,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Ldflags",
			Tag:          flag.TypeLinker,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Export_ldflags",
			Tag:          flag.TypeLinker,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Local_include_dirs",
			Tag:          flag.TypeIncludeLocal,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Export_local_include_dirs",
			Tag:          flag.TypeIncludeLocal,
			Factory:      flag.FromIncludePathOwned,
		},
		// For system includes, the path used to compile the current module uses `-I`,
		// the path to consumer modules will be using `-isystem` instead. For this reason `flag.TypeIncludeSystem`
		// is not present in this getter.
		{
			PropertyName: "Export_local_system_include_dirs",
			Tag:          flag.TypeIncludeLocal,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Include_dirs",
			Tag:          flag.TypeUnset,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Export_include_dirs",
			Tag:          flag.TypeUnset,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Export_system_include_dirs",
			Tag:          flag.TypeUnset,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Ldlibs",
			Tag:          flag.TypeLinkLibrary,
			Factory:      flag.FromStringOwned,
		},
	}
}

func (m *ModuleLibrary) getFlagOutLut() flag.FlagParserTable {
	return flag.FlagParserTable{
		{
			PropertyName: "Export_cflags",
			Tag:          flag.TypeCC | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Export_ldflags",
			Tag:          flag.TypeLinker | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Export_local_include_dirs",
			Tag:          flag.TypeIncludeLocal | flag.TypeExported,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Export_local_system_include_dirs",
			Tag:          flag.TypeIncludeLocal | flag.TypeExported | flag.TypeIncludeSystem,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Export_include_dirs",
			Tag:          flag.TypeExported,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Export_system_include_dirs",
			Tag:          flag.TypeExported | flag.TypeIncludeSystem,
			Factory:      flag.FromIncludePathOwned,
		},
		{
			PropertyName: "Ldlibs",
			Tag:          flag.TypeLinkLibrary | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
	}
}

func (m *ModuleLibrary) FlagsIn() flag.Flags {
	return flag.ParseFromProperties(nil, m.getFlagInLut(), m.Properties)
}

func (m *ModuleLibrary) FlagsInTransitive(ctx blueprint.BaseModuleContext) (ret flag.Flags) {
	// TODO: Local flags should take priority, they do not currently to match the pre-refactor behaviour.
	m.FlagsIn().ForEach(
		func(f flag.Flag) {
			ret = append(ret, f)
		})

	flag.ReferenceFlagsInTransitive(ctx).ForEach(
		func(f flag.Flag) {
			ret = append(ret, f)
		})
	return
}

func (m *ModuleLibrary) FlagsOut() flag.Flags {
	return flag.ParseFromProperties(nil, m.getFlagOutLut(), m.Properties)
}

func (m *ModuleLibrary) FlagsOutTargets() []string {
	return append(m.Properties.Reexport_libs, m.Properties.Export_generated_headers...)
}

func (m *ModuleLibrary) getLegacySourceProperties() *LegacySourceProps {
	return &m.Properties.LegacySourceProps
}

func (m *ModuleLibrary) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleLibrary) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.GetFiles(ctx)
}

func (m *ModuleLibrary) GetDirectFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleLibrary) GetTargets() (tgts []string) {
	tgts = append(tgts, m.Properties.GetTargets()...)
	tgts = append(tgts, m.Properties.Generated_sources...)
	return
}

// {{match_srcs}} template is only applied in specific properties where we've
// seen sensible use-cases and for `BuildProps` this is:
//   - Ldflags
//   - Cflags
//   - Conlyflags
//   - Cxxflags
func (m *ModuleLibrary) getMatchSourcePropNames() []string {
	return []string{"Ldflags", "Cflags", "Conlyflags", "Cxxflags"}
}

// Returns the shortname for the output, which is used as a phony target. If it
// can be built for multiple variants, require a '__host' or '__target' suffix to
// disambiguate.
func (m *ModuleLibrary) shortName() string {
	if len(m.supportedVariants()) > 1 {
		return m.Name() + "__" + string(m.Properties.TargetType)
	}
	return m.Name()
}

func GetGeneratedHeadersFiles(ctx blueprint.ModuleContext) (orderOnly []string) {
	visited := map[string]bool{}
	root := ctx.Module()
	ctx.WalkDeps(func(child, parent blueprint.Module) bool {
		childTag := ctx.OtherModuleDependencyTag(child)
		/* We want all the export_gen_include_dirs from generated modules mentioned by the
		 * main module, primarily from generated_headers, but also static_libs and
		 * shared_libs where they refer to a bob_generated_[static|shared]_library.
		 *
		 * We also want all the export_generated_headers from libraries mentioned by the main
		 * module, i.e. from static_libs and shared_libs, as well as
		 * export_generated_headers from the main module itself.
		 *
		 * Note that generated_header and export_generated_header tags can't have child
		 * generated_header, export_generated_header, static_libs or shared_libs tags,
		 * because these are only added by libraries.
		 */
		importHeaderDirs := false
		visitChildren := false
		if parent == root {
			if childTag == tag.GeneratedHeadersTag || childTag == tag.ExportGeneratedHeadersTag {
				importHeaderDirs = true
				visitChildren = false
			} else if childTag == tag.StaticTag || childTag == tag.SharedTag || childTag == tag.ReexportLibraryTag {
				/* Try to import generated header dirs from static|shared_libs too:
				 * - The library could be a bob_generate_shared_library or
				 *   bob_generate_static_library, in which case we need to import
				 *   any generated header dirs it exports.
				 * - If it's a bob_static_library or bob_shared_library, it may
				 *   export generated header dirs, so it's children need visiting.
				 */
				importHeaderDirs = true
				visitChildren = true
			}
		} else {
			if childTag == tag.ExportGeneratedHeadersTag {
				importHeaderDirs = true
				visitChildren = false
			}
		}
		if importHeaderDirs {
			if _, seen := visited[child.Name()]; !seen {
				visited[child.Name()] = true
				// Generated headers are "order-only". That means that a source file does not need to rebuild
				// if a generated header changes, just that it must be built after a generated header.
				// The source file _will_ be rebuilt if it uses the header (since that is registered in the
				// depfile). Note that this means that generated headers cannot change which headers are used
				// (by aliasing another header).
				if provider, ok := child.(file.Provider); ok {
					provider.OutFiles().ForEachIf(
						func(fp file.Path) bool {
							return (fp.IsType(file.TypeGenerated) || fp.IsType(file.TypeImplicit)) &&
								fp.IsNotType(file.TypeRsp) && fp.IsNotType(file.TypeDep) && fp.IsNotType(file.TypeToc)
						},
						func(fp file.Path) bool {
							orderOnly = append(orderOnly, fp.BuildPath())
							return true
						})
				}
			}
		}
		return visitChildren
	})
	return
}

func (m *ModuleLibrary) GetBuildWrapperAndDeps(ctx blueprint.ModuleContext) (string, []string) {
	return m.Properties.Build.GetBuildWrapperAndDeps(ctx)
}

func (m *ModuleLibrary) GetMteProps(blueprint.ModuleContext) AndroidMTEProps {
	return m.Properties.AndroidMTEProps
}

func (m *ModuleLibrary) IsHwAsanEnabled() bool {
	return proptools.Bool(m.Properties.Build.Hwasan_enabled)
}

func (m *ModuleLibrary) IsForwardingSharedLibrary() bool {
	return m.Properties.isForwardingSharedLibrary()
}

func (m *ModuleLibrary) IsRpathWanted() bool {
	return m.Properties.isRpathWanted()
}

func (m *ModuleLibrary) getAllGeneratedSourceModules(ctx blueprint.ModuleContext) (modules []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.GeneratedSourcesTag },
		func(m blueprint.Module) {
			if gs, ok := getGenerateCommon(m); ok {
				// Add our own name
				modules = append(modules, gs.Name())
			}
		})
	return
}

func (m *ModuleLibrary) getVersionScript(ctx blueprint.ModuleContext) *string {
	if m.Properties.VersionScriptModule != nil {
		module, _ := ctx.GetDirectDep(*m.Properties.VersionScriptModule)
		outputs := file.GetOutputs(module.(dependentInterface))
		if len(outputs) != 1 {
			panic(errors.New(ctx.OtherModuleName(module) + " must have exactly one output"))
		}
		return &outputs[0]
	}

	if m.Properties.Build.Version_script != nil {
		path := getBackendPathInSourceDir(getGenerator(ctx), *m.Properties.Build.Version_script)
		return &path
	}

	return nil
}

func (m *ModuleLibrary) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.Build.processPaths(ctx)

	versionScript := m.Properties.Build.Version_script
	if versionScript != nil {
		matches := depOutputsVarRegexp.FindStringSubmatch(*versionScript)
		if len(matches) == 2 {
			m.Properties.VersionScriptModule = &matches[1]
		} else {
			*versionScript = filepath.Join(projectModuleDir(ctx), *versionScript)
		}
	}
}

func (m *ModuleLibrary) checkField(cond bool, fieldName string) {
	if !cond {
		utils.Die("%s has field %s set", m.Name(), fieldName)
	}
}

func (m *ModuleLibrary) exportSharedLibs() []string { return m.Properties.Shared_libs }

func (m *ModuleLibrary) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleLibrary) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleLibrary) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleLibrary) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

func (m *ModuleLibrary) LibraryFactory(config *BobConfig, module blueprint.Module) (blueprint.Module, []interface{}) {
	m.Properties.Features.Init(&config.Properties,
		CommonProps{},
		BuildProps{},
		SplittableProps{},
		TagableProps{},
		ExternalableProps{})
	m.Properties.Host.init(&config.Properties,
		CommonProps{},
		BuildProps{},
		TagableProps{})
	m.Properties.Target.init(&config.Properties,
		CommonProps{},
		BuildProps{},
		TagableProps{})

	return module, []interface{}{&m.Properties, &m.SimpleName.Properties}
}

func (m *ModuleLibrary) GenerateBuildActions(blueprint.ModuleContext) {
	// Stub to fullfill blueprint.Module
}

func getBinaryOrSharedLib(m blueprint.Module) (*ModuleLibrary, bool) {
	if sl, ok := m.(*ModuleSharedLibrary); ok {
		return &sl.ModuleLibrary, true
	} else if b, ok := m.(*ModuleBinary); ok {
		return &b.ModuleLibrary, true
	}

	return nil, false
}

func getLibrary(m blueprint.Module) (*ModuleLibrary, bool) {
	if bsl, ok := getBinaryOrSharedLib(m); ok {
		return bsl, true
	} else if sl, ok := m.(*ModuleStaticLibrary); ok {
		return &sl.ModuleLibrary, true
	}

	return nil, false
}

func checkLibraryFieldsMutator(ctx blueprint.BottomUpMutatorContext) {
	m := ctx.Module()
	if b, ok := m.(*ModuleBinary); ok {
		props := b.Properties
		b.checkField(len(props.Export_cflags) == 0, "export_cflags")
		b.checkField(len(props.Export_include_dirs) == 0, "export_include_dirs")
		b.checkField(len(props.Export_ldflags) == 0, "export_ldflags")
		b.checkField(len(props.Export_local_include_dirs) == 0, "export_local_include_dirs")
		b.checkField(len(props.Export_local_system_include_dirs) == 0, "export_local_system_include_dirs")
		b.checkField(len(props.Export_system_include_dirs) == 0, "export_system_include_dirs")
		b.checkField(len(props.Reexport_libs) == 0, "reexport_libs")
		b.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
	} else if sl, ok := m.(*ModuleSharedLibrary); ok {
		props := sl.Properties
		if !sl.isExternal() {
			sl.checkField(len(props.Export_ldflags) == 0, "export_ldflags")
		}
		sl.checkField(props.Mte.Memtag_heap == nil, "memtag_heap")
		sl.checkField(props.Mte.Diag_memtag_heap == nil, "memtag_heap")
	} else if sl, ok := m.(*ModuleStaticLibrary); ok {
		props := sl.Properties
		sl.checkField(props.Forwarding_shlib == nil, "forwarding_shlib")
		sl.checkField(props.Version_script == nil, "version_script")
		sl.checkField(props.Mte.Memtag_heap == nil, "memtag_heap")
		sl.checkField(props.Mte.Diag_memtag_heap == nil, "memtag_heap")
	}
}

// Check that each module only reexports libraries that it is actually using.
func checkReexportLibsMutator(ctx blueprint.TopDownMutatorContext) {
	if l, ok := getLibrary(ctx.Module()); ok {
		for _, lib := range l.Properties.Reexport_libs {
			if !utils.ListsContain(lib,
				l.Properties.Shared_libs,
				l.Properties.Static_libs,
				l.Properties.Header_libs,
				l.Properties.Whole_static_libs,
				l.Properties.Export_header_libs) {
				utils.Die("%s re-exports unused library %s", ctx.ModuleName(), lib)
			}
		}
	}
}

// Traverse the dependency tree, following all StaticDepTag and WholeStaticDepTag links.
// Do *not* include modules which are in the tree via any other dependency tag.
func getLinkableModules(ctx blueprint.TopDownMutatorContext) map[blueprint.Module]bool {
	ret := make(map[blueprint.Module]bool)

	ctx.WalkDeps(func(dep blueprint.Module, parent blueprint.Module) bool {
		// Stop iteration once we get to other kinds of dependency which won't
		// actually be linked.
		if ctx.OtherModuleDependencyTag(dep) != tag.StaticTag &&
			ctx.OtherModuleDependencyTag(dep) != tag.WholeStaticTag {
			return false
		}
		ret[dep] = true

		return true
	})

	return ret
}

// Check that no libraries are being accidentally linked twice, by having one copy
// linked explicitly (via static_libs), and another included in a different
// library via whole_static_libs.
func checkForMultipleLinking(topLevelModuleName string, staticLibs map[string]bool, insideWholeLibs map[string]string) {
	duplicateDeps := []string{}
	for dep := range staticLibs {
		if _, ok := insideWholeLibs[dep]; ok {
			duplicateDeps = append(duplicateDeps, dep)
		}
	}

	if len(duplicateDeps) > 0 {
		msg := fmt.Sprintf("Warning: %s links with the following libraries multiple times:\n", topLevelModuleName)
		for _, dep := range duplicateDeps {
			msg += fmt.Sprintf("  * %s, but also %s, which includes %s as a whole_static_lib\n",
				dep, insideWholeLibs[dep], dep)
		}
		utils.Die(msg)
	}
}

type SharedLibraryExporter interface {
	flag.Provider // Eventually the below functions will be removed
	exportSharedLibs() []string
}

// While traversing the static library dependency tree, propagate extra properties.
func propagateOtherExportedProperties(m *ModuleLibrary, depLib SharedLibraryExporter) {
	props := &m.Properties.Build
	for _, shLib := range depLib.exportSharedLibs() {
		if !utils.Contains(props.Shared_libs, shLib) {
			props.Shared_libs = append(props.Shared_libs, shLib)
			props.ExtraSharedLibs = append(props.ExtraSharedLibs, shLib)
		}
	}

	depLib.FlagsOut().Filtered(func(f flag.Flag) bool {
		return f.MatchesType(flag.TypeLinkLibrary)
	}).ForEachIf(
		func(f flag.Flag) bool { return !utils.Contains(props.Ldlibs, f.ToString()) },
		func(f flag.Flag) {
			props.Ldlibs = append(props.Ldlibs, f.ToString())
		})

	props.Ldflags = append(props.Ldflags, depLib.FlagsOut().Filtered(func(f flag.Flag) bool {
		return f.MatchesType(flag.TypeLinker)
	}).ToStringSlice()...)

	// Header libraries are *not* propagated here, because they are currently
	// only supported on Android, which will automatically re-export them just
	// by adding them to LOCAL_EXPORT_HEADER_LIBRARY_HEADERS.
}

func exportLibFlagsMutator(ctx blueprint.TopDownMutatorContext) {
	l, ok := getBinaryOrSharedLib(ctx.Module())
	if !ok {
		return
	}

	// Track the set of everything mentioned in 'static_libs' of all
	// dependencies of this module, for multiple-link checking.
	allImportedStaticLibs := make(map[string]bool)
	// Map between a library name and the first encountered lib in which it
	// is used in whole_static_libs.
	insideWholeLibs := make(map[string]string)
	// VisitDepsDepthFirst doesn't let us stop iteration, so get the list of
	// modules to examine separately using WalkDeps.
	modulesToVisit := getLinkableModules(ctx)

	ctx.VisitDepsDepthFirst(func(dep blueprint.Module) {
		if _, ok := modulesToVisit[dep]; !ok {
			return
		}

		if depLib, ok := dep.(*ModuleStaticLibrary); ok {
			if !depLib.isExternal() {
				// TODO: whole static libs should use a tag with relevant information.
				for _, subLib := range depLib.Properties.Whole_static_libs {
					if firstContainingLib, ok := insideWholeLibs[subLib]; ok {
						utils.Die("%s links with %s and %s, which both contain %s as whole_static_libs",
							ctx.Module().Name(), firstContainingLib,
							depLib.Name(), subLib)
					} else {
						insideWholeLibs[subLib] = depLib.Name()
					}
				}
				for _, subLib := range depLib.Properties.Static_libs {
					allImportedStaticLibs[subLib] = true
				}
			}

			propagateOtherExportedProperties(l, depLib)
		} else if _, ok := dep.(*generateStaticLibrary); ok {
			// Nothing to do for GeneratedStaticLibrary
			//
			// The GeneratedStaticLibrary is expected to be self
			// contained, so no pulling in of other static or shared
			// libraries.
		} else if depLib, ok := dep.(*ModuleExternalLibrary); ok {
			propagateOtherExportedProperties(l, depLib)
		} else if _, ok := dep.(*ModuleStrictLibrary); ok {
			// TODO: Propogate flags here?
		} else {
			utils.Die("%s is not a staticLibrary", dep.Name())
		}

		// Don't add whole_static_lib components to the library list, because their
		// contents are already included in the parent library.
		if ctx.OtherModuleDependencyTag(dep) != tag.WholeStaticTag &&
			ctx.OtherModuleDependencyTag(dep) != tag.StaticTag &&
			ctx.OtherModuleDependencyTag(dep) != tag.DepTag {
			utils.Die("Non WholeStatic or Static dep tag encountered visiting %s from %s",
				dep.Name(), ctx.ModuleName())
		}
	})

	checkForMultipleLinking(ctx.ModuleName(), allImportedStaticLibs, insideWholeLibs)
}
