package core

import (
	"path/filepath"
	"regexp"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
)

// EnableableProps allow a module to be disabled or only built when explicitly requested
type EnableableProps struct {
	// Used to disable the generation of build rules. If this is set to false, no build rule will be generated.
	Enabled *bool
	// Whether it is built by default in a build with no targets requested.
	// Nothing to do with 'defaults'.
	Build_by_default *bool
	// Is this module depended on by a module which is built by default?
	// Used to prune unused modules from Android builds, where we can't
	// control exactly what gets built.
	Required bool `blueprint:"mutated"`
}

// Modules implementing the enableable interface can be disabled, and select if they are built by default
type enableable interface {
	getEnableableProps() *EnableableProps
}

func isEnabled(e enableable) bool {
	props := e.getEnableableProps()
	if props.Enabled != nil {
		return *props.Enabled
	}
	return true
}

func isBuiltByDefault(e enableable) bool {
	props := e.getEnableableProps()
	if props.Build_by_default != nil {
		return *props.Build_by_default
	}

	switch m := e.(type) {
	case *ModuleBinary, *ModuleStrictBinary:
		var tgt toolchain.TgtType = toolchain.TgtTypeUnknown

		if m, ok := m.(splittable); ok {
			tgt = m.getTarget()
		}

		return tgt == toolchain.TgtTypeTarget
	case *ModuleKernelObject:
		return true
	}
	return false
}

func isRequired(e enableable) bool {
	return e.getEnableableProps().Required
}

func markAsRequired(e enableable) {
	e.getEnableableProps().Required = true
}

// InstallableProps are embedded by modules which can be installed outside the
// build directory
type InstallableProps struct {
	// Module specifying an installation directory
	Install_group *string
	// Other modules which must be installed alongside this
	Install_deps []string
	// Path to install to, relative to the install_group's path
	Relative_install_path *string
	// Script used during post install
	Post_install_tool *string
	// Command to execute on file(s) after they are installed
	Post_install_cmd *string
	// Arguments to post install command
	Post_install_args []string
	// The path retrieved from the install group so we don't need to walk dependencies to get it
	InstallGroupPath *string `blueprint:"mutated"`
}

func (props *InstallableProps) processPaths(ctx blueprint.BaseModuleContext) {
	if props.Post_install_tool != nil {
		*props.Post_install_tool = getBackendPathInSourceDir(getGenerator(ctx), projectModuleDir(ctx), *props.Post_install_tool)
	}
}

func (props *InstallableProps) getInstallPath() (string, bool) {
	if props.InstallGroupPath == nil {
		return "", false
	}

	installPath := *props.InstallGroupPath

	if props.Relative_install_path != nil {
		installPath = filepath.Join(installPath, *props.Relative_install_path)
	}

	return installPath, true
}

func getShortNamesForDirectDepsIf(ctx blueprint.ModuleContext,
	pred func(m blueprint.Module) bool) (ret []string) {

	visited := map[string]bool{}

	ctx.VisitDirectDepsIf(pred,
		func(m blueprint.Module) {
			if dep, ok := m.(phonyInterface); ok {
				if _, ok := visited[m.Name()]; !ok {
					ret = append(ret, dep.shortName())
				}
			} else {
				utils.Die("install_dep on non-dependendable module %s", m.Name())
			}
			visited[m.Name()] = true
		})
	return
}

func getShortNamesForDirectDepsWithTags(ctx blueprint.ModuleContext,
	tags ...tag.DependencyTag) (ret []string) {

	return getShortNamesForDirectDepsIf(ctx,
		func(m blueprint.Module) bool {
			tag := ctx.OtherModuleDependencyTag(m)
			for _, i := range tags {
				if tag == i {
					return true
				}
			}
			return false
		})
}

func getShortNamesForDirectDepsWithTagsForNonFilegroup(ctx blueprint.ModuleContext,
	tags ...tag.DependencyTag) (ret []string) {

	return getShortNamesForDirectDepsIf(ctx,
		func(m blueprint.Module) bool {
			tag := ctx.OtherModuleDependencyTag(m)

			// Do not count `ModuleFilegroup` as dependency.
			// `ModuleFilegroup` are specified by `tag.GeneratedTag`
			// dependency tag but they are simple file providers
			// and cannot be considered as `generated_deps`.
			if _, ok := m.(*ModuleFilegroup); ok {
				return false
			}

			for _, i := range tags {
				if tag == i {
					return true
				}
			}
			return false
		})
}

// InstallGroupProps describes the properties of bob_install_group modules
type InstallGroupProps struct {
	Install_path *string
}

type ModuleInstallGroup struct {
	module.ModuleBase
	Properties struct {
		InstallGroupProps
		TagableProps
		Features
	}
}

func (m *ModuleInstallGroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// No build actions for a bob_install_group
}

func (m *ModuleInstallGroup) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.InstallGroupProps, &m.Properties.TagableProps}
}

func (m *ModuleInstallGroup) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleInstallGroup) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleInstallGroup) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleInstallGroup) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleInstallGroup) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

// Modules implementing the installable interface can be install their output
type installable interface {
	file.Provider
	getInstallableProps() *InstallableProps
	getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string
}

// ResourceProps defines all the properties that can be used in a bob_resource
type ResourceProps struct {
	LegacySourceProps
	AliasableProps
	InstallableProps
	EnableableProps
	AndroidProps
	TagableProps
}

type ModuleResource struct {
	module.ModuleBase
	Properties struct {
		ResourceProps
		Features
	}
}

type resourceInterface interface {
	pathProcessor
	file.Resolver
	file.Consumer
	file.Provider
	Tagable
}

var _ resourceInterface = (*ModuleResource)(nil) // impl check

func (m *ModuleResource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).resourceActions(m, ctx)
	}
}

func (m *ModuleResource) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.ResourceProps}
}

func (m *ModuleResource) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleResource) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, tag.InstallTag)
}

func (m *ModuleResource) shortName() string {
	return m.Name()
}

func (m *ModuleResource) altName() string {
	return m.Name()
}

func (m *ModuleResource) altShortName() string {
	return m.shortName()
}

func (m *ModuleResource) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *ModuleResource) OutFiles() (files file.Paths) {
	m.Properties.LegacySourceProps.GetDirectFiles().ForEach(
		func(fp file.Path) bool {
			files = append(files, file.FromWithTag(&fp, file.TypeInstallable))
			return true
		},
	)
	return
}

func (m *ModuleResource) OutFileTargets() []string { return []string{} }

func (m *ModuleResource) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *ModuleResource) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.LegacySourceProps.processPaths(ctx)
	m.Properties.InstallableProps.processPaths(ctx)
}

func (m *ModuleResource) GetTargets() []string {
	return m.Properties.LegacySourceProps.GetTargets()
}

func (m *ModuleResource) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.Properties.LegacySourceProps.GetFiles(ctx)
}

func (m *ModuleResource) GetDirectFiles() file.Paths {
	return m.Properties.LegacySourceProps.GetDirectFiles()
}

func (m *ModuleResource) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleResource) getAliasList() []string {
	return m.Properties.getAliasList()
}

func (m ModuleInstallGroup) GetProperties() interface{} {
	return m.Properties
}

func (m ModuleResource) GetProperties() interface{} {
	return m.Properties
}

func (m *ModuleResource) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleResource) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleResource) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleResource) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

func installGroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleInstallGroup{}
	module.Properties.Features.Init(&config.Properties, InstallGroupProps{}, TagableProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

func resourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleResource{}
	module.Properties.Features.Init(&config.Properties, ResourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

func getInstallGroupPathFromTag(ctx blueprint.TopDownMutatorContext, inputTag tag.DependencyTag) *string {
	var installGroupPath *string

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == inputTag },
		func(m blueprint.Module) {
			insg, ok := m.(*ModuleInstallGroup)
			if !ok {
				utils.Die("%s dependency of %s not an install group",
					inputTag.Name, ctx.ModuleName())
			}
			if installGroupPath != nil {
				utils.Die("Multiple %s dependencies for %s",
					inputTag.Name, ctx.ModuleName())
			}
			installGroupPath = insg.Properties.Install_path
		})

	return installGroupPath
}

func installGroupMutator(ctx blueprint.TopDownMutatorContext) {
	if ins, ok := ctx.Module().(installable); ok {
		path := getInstallGroupPathFromTag(ctx, tag.InstallGroupTag)
		if path != nil {
			if *path == "" {
				utils.Die("Module %s has empty install path", ctx.ModuleName())
			}

			props := ins.getInstallableProps()
			props.InstallGroupPath = path
		}
	}
}
