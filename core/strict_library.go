package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
)

// In Bazel, some properties are transitive.
type TransitiveLibraryProps struct {
	Defines []string
}

func (m *TransitiveLibraryProps) defines() []string {
	return m.Defines
}

type StrictLibraryProps struct {
	Hdrs []string
	// TODO: Header inclusion
	//Textual_hdrs           []string
	//Includes               []string
	//Include_prefixes       []string
	//Strip_include_prefixes []string

	Local_defines []string
	Copts         []string
	Deps          []string

	// TODO: unused but needed for the output interface, no easy way to hide it
	Out *string

	TargetType toolchain.TgtType `blueprint:"mutated"`
}

// List of include dirs to be added to the compile line.
//
// Each string is prepended with `-I` when building the target itself
// and `-isystem` when building modules who consumes it.
// Unlike `Copts`, these flags are added for this rule and every
// rule that depends on it.
type IncludeProps struct {
	Includes []string
}

type ModuleStrictLibrary struct {
	module.ModuleBase
	Properties struct {
		StrictLibraryProps
		SourceProps
		TransitiveLibraryProps
		Features
		EnableableProps
		SplittableProps
		InstallableProps
		IncludeProps
	}
}

type strictLibraryInterface interface {
	splittable
	dependentInterface
	FileConsumer
	FileResolver
}

var _ strictLibraryInterface = (*ModuleStrictLibrary)(nil)

func (m *ModuleStrictLibrary) processPaths(ctx blueprint.BaseModuleContext) {
	// TODO: Handle Bazel targets & check paths
	prefix := projectModuleDir(ctx)
	m.Properties.SourceProps.processPaths(ctx)
	m.Properties.Hdrs = utils.PrefixDirs(m.Properties.Hdrs, prefix)
}

func (m *ModuleStrictLibrary) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	return m.OutFiles().ToStringSliceIf(
		func(p file.Path) bool {
			return p.IsType(file.TypeArchive) ||
				p.IsType(file.TypeShared)
		},
		func(p file.Path) string {
			return p.BuildPath()
		})
}

func (m *ModuleStrictLibrary) outputName() string {
	if m.Properties.Out != nil {
		return *m.Properties.Out
	}
	return m.Name()
}

func (m *ModuleStrictLibrary) outputFileName() string {
	utils.Die("Cannot use outputFileName on strict_library")
	return "badName"
}

func (m *ModuleStrictLibrary) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.GetFiles(ctx)
}

func (m *ModuleStrictLibrary) GetDirectFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleStrictLibrary) GetTargets() (tgts []string) {
	tgts = append(tgts, m.Properties.GetTargets()...)
	return
}

func (m *ModuleStrictLibrary) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleStrictLibrary) implicitOutputs() []string {
	return []string{}
}

func (m *ModuleStrictLibrary) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return []string{}
}

func (m *ModuleStrictLibrary) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		// TODO: fixme, for now shared outputs are not supported
		func(f file.Path) bool { return f.IsType(file.TypeArchive) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleStrictLibrary) OutFiles() file.Paths {
	return file.Paths{
		file.NewPath(m.Name()+".a", string(m.getTarget()), file.TypeArchive),
		file.NewPath(m.Name()+".so", string(m.getTarget()), file.TypeShared),
	}
}

func (m *ModuleStrictLibrary) OutFileTargets() []string {
	return []string{}
}

func (m *ModuleStrictLibrary) FlagsIn() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Copts",
			Tag:          flag.TypeCC,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Local_defines",
			Tag:          flag.TypeUnset,
			Factory:      flag.FromDefineOwned,
		},
		{
			PropertyName: "Defines",
			Tag:          flag.TypeUnset,
			Factory:      flag.FromDefineOwned,
		},
		{
			PropertyName: "Includes",
			Tag:          flag.TypeInclude,
			Factory:      flag.FromIncludePathOwned,
		},
	}

	return flag.ParseFromProperties(nil, lut, m.Properties)
}

func (m *ModuleStrictLibrary) FlagsInTransitive(ctx blueprint.BaseModuleContext) (ret flag.Flags) {
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

func (m *ModuleStrictLibrary) FlagsOut() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Defines",
			Tag:          flag.TypeExported | flag.TypeTransitive,
			Factory:      flag.FromDefineOwned,
		},
		{
			PropertyName: "Includes",
			Tag:          flag.TypeExported | flag.TypeTransitive | flag.TypeIncludeSystem,
			Factory:      flag.FromIncludePathOwned,
		},
	}

	return flag.ParseFromProperties(nil, lut, m.Properties)
}

func (m *ModuleStrictLibrary) supportedVariants() (tgts []toolchain.TgtType) {
	// TODO: Change tgts based on if host or target supported.
	tgts = append(tgts, toolchain.TgtTypeHost)
	return
}

func (m *ModuleStrictLibrary) disable() {
	f := false
	m.Properties.Enabled = &f
}

func (m *ModuleStrictLibrary) setVariant(tgt toolchain.TgtType) {
	m.Properties.TargetType = tgt
}

func (m *ModuleStrictLibrary) getTarget() toolchain.TgtType {
	return m.Properties.TargetType
}

func (m *ModuleStrictLibrary) getSplittableProps() *SplittableProps {
	return &m.Properties.SplittableProps
}

func (m *ModuleStrictLibrary) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *ModuleStrictLibrary) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *ModuleStrictLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).strictLibraryActions(m, ctx)
}

func (m *ModuleStrictLibrary) shortName() string {
	return m.Name()
}

// Shared Library BoB Interface

func (m *ModuleStrictLibrary) getTocName() string {
	// TODO: Does this need to be m.getRealName() It is in other impls
	// what does getRealName() look like?
	return m.Name() + tocExt
}

func (m ModuleStrictLibrary) GetProperties() interface{} {
	return m.Properties
}

func (m *ModuleStrictLibrary) GetBuildWrapperAndDeps(ctx blueprint.ModuleContext) (string, []string) {
	return "", []string{}
}

func (m *ModuleStrictLibrary) GetStaticLibs(ctx blueprint.ModuleContext) (libs []string) {
	// Required for legacy backend implementation
	return
}

func (m *ModuleStrictLibrary) IsForwardingSharedLibrary() bool {
	// Required for legacy backend implementation
	// Forwarding not supported yet
	return false
}

func (m *ModuleStrictLibrary) IsRpathWanted() bool {
	// Required for legacy backend implementation
	// Rpath is not supported
	return false
}

func (m *ModuleStrictLibrary) getLinkName() string {
	return m.outputName() + ".so"
}

func (m *ModuleStrictLibrary) getRealName() string {
	// Required for legacy backend implementation
	return m.getLinkName()
}

func (m *ModuleStrictLibrary) getVersionScript(ctx blueprint.ModuleContext) *string {
	// Required for legacy backend implementation
	// Versioning not yet supported
	return nil
}

func LibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleStrictLibrary{}
	module.Properties.Features.Init(&config.Properties, StrictLibraryProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

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
