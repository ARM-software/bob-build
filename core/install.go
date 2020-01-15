/*
 * Copyright 2018-2020 Arm Limited.
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
	"fmt"
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/abstr"
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
	case *binary:
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
	Install_path *string `blueprint:"mutated"`
}

func (props *InstallableProps) processPaths(ctx abstr.BaseModuleContext, g generatorBackend) {
	if props.Post_install_tool != nil {
		*props.Post_install_tool = filepath.Join(g.sourcePrefix(), projectModuleDir(ctx), *props.Post_install_tool)
	}
}

func (props *InstallableProps) getInstallGroupPath() (path string, ok bool) {
	if props.Install_path == nil {
		return "", false
	}
	return proptools.String(props.Install_path), true
}

func getShortNamesForDirectDepsWithTags(ctx blueprint.ModuleContext,
	tags ...dependencyTag) (ret []string) {
	visited := map[string]bool{}
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool {
			tag := ctx.OtherModuleDependencyTag(m)
			for _, i := range tags {
				if tag == i {
					return true
				}
			}
			return false
		},
		func(m blueprint.Module) {
			if dep, ok := m.(phonyInterface); ok {
				if _, ok := visited[m.Name()]; !ok {
					ret = append(ret, dep.shortName())
				}
			} else {
				panic("install_dep on non-dependendable module")
			}
			visited[m.Name()] = true
		})
	return
}

// InstallGroupProps describes the properties of bob_install_group modules
type InstallGroupProps struct {
	Install_path string
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

func (m *installGroup) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.InstallGroupProps}
}

func (m *installGroup) features() *Features {
	return &m.Properties.Features
}

// Modules implementing the symlinkInstaller interface are able to create symlinks in the install location
type symlinkInstaller interface {
	librarySymlinks(ctx blueprint.ModuleContext) map[string]string
}

// Modules implementing the installable interface can be install their output
type installable interface {
	filesToInstall(ctx abstr.BaseModuleContext, g generatorBackend) []string
	getInstallableProps() *InstallableProps
	getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string
}

// ResourceProps defines all the properties that can be used in a bob_resource
type ResourceProps struct {
	SourceProps
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

func (m *resource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getBackend(ctx).resourceActions(m, ctx)
	}
}

func (m *resource) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.ResourceProps}
}

func (m *resource) features() *Features {
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

// Resources don't have any outputs (i.e. stuff generated in the build
// directory) - they only copy source files to the installation dir. This
// method exists to implement PhonyInterface.
func (m *resource) outputs(g generatorBackend) []string {
	return []string{}
}

func (m *resource) implicitOutputs(g generatorBackend) []string {
	return []string{}
}

func (m *resource) filesToInstall(ctx abstr.BaseModuleContext, g generatorBackend) []string {
	return m.Properties.SourceProps.getSources(ctx)
}

func (m *resource) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *resource) processPaths(ctx abstr.BaseModuleContext, g generatorBackend) {
	m.Properties.SourceProps.processPaths(ctx, g)
	m.Properties.InstallableProps.processPaths(ctx, g)
}

func (m *resource) getAliasList() []string {
	return m.Properties.getAliasList()
}

func installGroupFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &installGroup{}
	module.Properties.Features.Init(&config.Properties, InstallGroupProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

func resourceFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &resource{}
	module.Properties.Features.Init(&config.Properties, ResourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

var installGroupTag = dependencyTag{name: "install_group"}
var installDepTag = dependencyTag{name: "install_dep"}

func getInstallPath(mctx abstr.TopDownMutatorContext, tag dependencyTag) *string {
	var installGroupPath *string

	abstr.VisitDirectDepsIf(
		mctx,
		func(m blueprint.Module) bool { return mctx.OtherModuleDependencyTag(m) == tag },
		func(m blueprint.Module) {
			insg, ok := m.(*installGroup)
			if !ok {
				panic(fmt.Sprintf("%s dependency of %s not an install group",
					tag.name, mctx.ModuleName()))
			}
			if installGroupPath != nil {
				panic(fmt.Sprintf("Multiple %s dependencies for %s",
					tag.name, mctx.ModuleName()))
			}
			installGroupPath = &insg.Properties.Install_path
		})

	return installGroupPath
}

func installGroupMutator(mctx abstr.TopDownMutatorContext) {
	if ins, ok := abstr.Module(mctx).(installable); ok {
		path := getInstallPath(mctx, installGroupTag)
		if path != nil {
			if *path == "" {
				panic(fmt.Sprintf("Module %s has empty install path", mctx.ModuleName()))
			}

			props := ins.getInstallableProps()
			props.Install_path = path
		}
	}
}
