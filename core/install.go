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
	"path/filepath"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
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
	case *ModuleBinary:
		if m.Properties.TargetType == toolchain.TgtTypeTarget {
			return true
		}
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
	tags ...DependencyTag) (ret []string) {

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
	tags ...DependencyTag) (ret []string) {

	return getShortNamesForDirectDepsIf(ctx,
		func(m blueprint.Module) bool {
			tag := ctx.OtherModuleDependencyTag(m)

			// Do not count `ModuleFilegroup` as dependency.
			// `ModuleFilegroup` are specified by `GeneratedTag`
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
		Features
	}
}

func (m *ModuleInstallGroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// No build actions for a bob_install_group
}

func (m *ModuleInstallGroup) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.InstallGroupProps}
}

func (m *ModuleInstallGroup) Features() *Features {
	return &m.Properties.Features
}

// Modules implementing the installable interface can be install their output
type installable interface {
	filesToInstall(ctx blueprint.BaseModuleContext) []string
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
	FileResolver
	FileConsumer
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
	return getShortNamesForDirectDepsWithTags(ctx, InstallTag)
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

func (m *ModuleResource) filesToInstall(ctx blueprint.BaseModuleContext) (files []string) {
	m.Properties.LegacySourceProps.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			files = append(files, fp.BuildPath())
			return true
		})
	return
}

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

func installGroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleInstallGroup{}
	module.Properties.Features.Init(&config.Properties, InstallGroupProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

func resourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleResource{}
	module.Properties.Features.Init(&config.Properties, ResourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

func getInstallGroupPathFromTag(ctx blueprint.TopDownMutatorContext, tag DependencyTag) *string {
	var installGroupPath *string

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag },
		func(m blueprint.Module) {
			insg, ok := m.(*ModuleInstallGroup)
			if !ok {
				utils.Die("%s dependency of %s not an install group",
					tag.name, ctx.ModuleName())
			}
			if installGroupPath != nil {
				utils.Die("Multiple %s dependencies for %s",
					tag.name, ctx.ModuleName())
			}
			installGroupPath = insg.Properties.Install_path
		})

	return installGroupPath
}

func installGroupMutator(ctx blueprint.TopDownMutatorContext) {
	if ins, ok := ctx.Module().(installable); ok {
		path := getInstallGroupPathFromTag(ctx, InstallGroupTag)
		if path != nil {
			if *path == "" {
				utils.Die("Module %s has empty install path", ctx.ModuleName())
			}

			props := ins.getInstallableProps()
			props.InstallGroupPath = path
		}
	}
}
