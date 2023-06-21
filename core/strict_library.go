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
	"os"
	"path/filepath"

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

type ModuleStrictLibrary struct {
	module.ModuleBase
	simpleOutputProducer // band-aid so legacy don't complain the interface isn't implemented
	Properties           struct {
		StrictLibraryProps
		SourceProps
		TransitiveLibraryProps
		Features
		EnableableProps
		SplittableProps
		InstallableProps
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

func (m *ModuleStrictLibrary) ObjDir() string {
	return filepath.Join("${BuildDir}", string(m.Properties.TargetType), "objects", m.outputName()) + string(os.PathSeparator)
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

func (m *ModuleStrictLibrary) GetGeneratedHeaders(ctx blueprint.ModuleContext) (includeDirs []string, orderOnly []string) {
	return
}

func LibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleStrictLibrary{}
	module.Properties.Features.Init(&config.Properties, StrictLibraryProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
