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

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/utils"
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
		if m.Properties.TargetType == tgtTypeTarget {
			return true
		}
	case *kernelModule:
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

func (props *InstallableProps) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	if props.Post_install_tool != nil {
		*props.Post_install_tool = getBackendPathInSourceDir(g, projectModuleDir(ctx), *props.Post_install_tool)
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
	tags ...dependencyTag) (ret []string) {

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

// InstallGroupProps describes the properties of bob_install_group modules
type InstallGroupProps struct {
	Install_path *string
}

type installGroup struct {
	moduleBase
	Properties struct {
		InstallGroupProps
		Features
	}
}

func (m *installGroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// No build actions for a bob_install_group
}

func (m *installGroup) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.InstallGroupProps}
}

func (m *installGroup) Features() *Features {
	return &m.Properties.Features
}

// Modules implementing the symlinkInstaller interface are able to create symlinks in the install location
type symlinkInstaller interface {
	librarySymlinks(ctx blueprint.ModuleContext) map[string]string
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

type resource struct {
	moduleBase
	Properties struct {
		ResourceProps
		Features
	}
}

type resourceInterface interface {
	pathProcessor
	FileResolver
	SourceFileConsumer
}

var _ resourceInterface = (*resource)(nil) // impl check

func (m *resource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).resourceActions(m, ctx)
	}
}

func (m *resource) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.ResourceProps}
}

func (m *resource) Features() *Features {
	return &m.Properties.Features
}

func (m *resource) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, installDepTag)
}

func (m *resource) shortName() string {
	return m.Name()
}

func (m *resource) altName() string {
	return m.Name()
}

func (m *resource) altShortName() string {
	return m.shortName()
}

func (m *resource) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *resource) filesToInstall(ctx blueprint.BaseModuleContext) (files []string) {
	m.Properties.LegacySourceProps.GetSrcs(ctx).ForEach(
		func(fp filePath) bool {
			files = append(files, fp.buildPath())
			return true
		})
	return
}

func (m *resource) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *resource) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.LegacySourceProps.processPaths(ctx, g)
	m.Properties.InstallableProps.processPaths(ctx, g)
}

func (m *resource) GetSrcTargets() []string {
	return m.Properties.LegacySourceProps.GetSrcTargets()
}

func (m *resource) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	return m.Properties.LegacySourceProps.GetSrcs(ctx)
}

func (m *resource) GetDirectSrcs() FilePaths {
	return m.Properties.LegacySourceProps.GetDirectSrcs()
}

func (m *resource) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.ResolveFiles(ctx, g)
}

func (m *resource) getAliasList() []string {
	return m.Properties.getAliasList()
}

func (m installGroup) GetProperties() interface{} {
	return m.Properties
}

func (m resource) GetProperties() interface{} {
	return m.Properties
}

func installGroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &installGroup{}
	module.Properties.Features.Init(&config.Properties, InstallGroupProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

func resourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &resource{}
	module.Properties.Features.Init(&config.Properties, ResourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

var installGroupTag = dependencyTag{name: "install_group"}
var installDepTag = dependencyTag{name: "install_dep"}

func getInstallGroupPathFromTag(mctx blueprint.TopDownMutatorContext, tag dependencyTag) *string {
	var installGroupPath *string

	mctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return mctx.OtherModuleDependencyTag(m) == tag },
		func(m blueprint.Module) {
			insg, ok := m.(*installGroup)
			if !ok {
				utils.Die("%s dependency of %s not an install group",
					tag.name, mctx.ModuleName())
			}
			if installGroupPath != nil {
				utils.Die("Multiple %s dependencies for %s",
					tag.name, mctx.ModuleName())
			}
			installGroupPath = insg.Properties.Install_path
		})

	return installGroupPath
}

func installGroupMutator(mctx blueprint.TopDownMutatorContext) {
	if ins, ok := mctx.Module().(installable); ok {
		path := getInstallGroupPathFromTag(mctx, installGroupTag)
		if path != nil {
			if *path == "" {
				utils.Die("Module %s has empty install path", mctx.ModuleName())
			}

			props := ins.getInstallableProps()
			props.InstallGroupPath = path
		}
	}
}
