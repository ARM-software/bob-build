/*
 * Copyright 2018-2020, 2022-2023 Arm Limited.
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
	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/utils"
)

// Support generation of static and shared libraries
// This file declares the common properties and functions needed by both.

// GenerateLibraryProps contain the properties that are specific to generating libraries
type GenerateLibraryProps struct {
	// List of headers that are created (if any)
	Headers []string

	// Alternate output name, used for the file name and Android rules
	Out *string

	// List of implicit sources. Implicit sources are input files that do not get
	// mentioned on the command line, and are not specified in the explicit sources.
	Implicit_srcs []string

	// Implicit source files that should not be included. Use with care.
	Exclude_implicit_srcs []string

	ResolvedOut FilePaths `blueprint:"mutated"`
}

type generateLibrary struct {
	generateCommon
	Properties struct {
		GenerateLibraryProps
	}
}

// Verify that the following interfaces are implemented
var _ phonyInterface = (*generateLibrary)(nil)
var _ dependentInterface = (*generateLibrary)(nil)
var _ splittable = (*generateLibrary)(nil)
var _ installable = (*generateLibrary)(nil)
var _ SourceFileConsumer = (*generateLibrary)(nil)
var _ SourceFileProvider = (*generateLibrary)(nil)

// Modules implementing generateLibraryInterface support arbitrary commands
// that either produce a static library, shared library or binary.
type generateLibraryInterface interface {
	blueprint.Module
	dependentInterface
	SourceFileProvider
	ImplicitFileConsumer

	libExtension() string
	outputFileName() string
	getDepfile() (string, bool)
}

// Map sources to outputs. This function is primarily to support
// transformSource, so here we return a single element associating all
// inputs with all outputs. Implicit outputs must be passed in.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies to
// out, implicitOuts, depfile and rspfile. The output directory (if
// needed) needs to be added in by the backend specific
// GenerateBuildAction()
func generateLibraryInouts(m generateLibraryInterface, ctx blueprint.ModuleContext,
	g generatorBackend, implicitOuts []string) []inout {
	var io inout

	m.GetSrcs(ctx).ForEach(
		func(fp filePath) bool {
			io.in = append(io.in, fp.buildPath())
			return true
		})

	m.GetImplicits(ctx).ForEach(
		func(fp filePath) bool {
			io.implicitSrcs = append(io.implicitSrcs, fp.buildPath())
			return true
		})

	io.out = []string{m.outputFileName()}
	if depfile, ok := m.getDepfile(); ok {
		io.depfile = depfile
	}

	io.implicitOuts = implicitOuts
	return []inout{io}
}

func (m *generateLibrary) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	pmdir := projectModuleDir(ctx)
	m.Properties.Implicit_srcs = utils.PrefixDirs(m.Properties.Implicit_srcs, pmdir)
	m.Properties.Exclude_implicit_srcs = utils.PrefixDirs(m.Properties.Exclude_implicit_srcs, pmdir)
	m.generateCommon.processPaths(ctx, g)
}

//// Support generateLibraryInterface

func (m *generateLibrary) getImplicitSources(ctx blueprint.BaseModuleContext) []string {
	return glob(ctx, m.Properties.Implicit_srcs, m.Properties.Exclude_implicit_srcs)
}

func (m *generateLibrary) GetSrcs(ctx blueprint.BaseModuleContext) (srcs FilePaths) {
	gc, _ := getGenerateCommon(m)
	srcs = gc.Properties.LegacySourceProps.GetSrcs(ctx)
	return
}

func (m *generateLibrary) GetDirectSrcs() (srcs FilePaths) {
	gc, _ := getGenerateCommon(m)
	srcs = gc.Properties.LegacySourceProps.GetDirectSrcs()
	return
}

func (m *generateLibrary) GetImplicits(ctx blueprint.BaseModuleContext) (implicits FilePaths) {
	g := getBackend(ctx)
	for _, s := range m.getImplicitSources(ctx) {
		implicits = append(implicits, newSourceFilePath(s, ctx, g))
	}
	return
}

func (m *generateLibrary) GetSrcTargets() (tgts []string) {
	gc, _ := getGenerateCommon(m)
	tgts = append(tgts, gc.Properties.LegacySourceProps.GetSrcTargets()...)
	tgts = append(tgts, gc.Properties.Generated_sources...)
	return
}

func (m *generateLibrary) OutSrcs() FilePaths {
	return m.Properties.ResolvedOut
}

func (m *generateLibrary) OutSrcTargets() []string {
	return []string{}
}

//// Support splittable

func (m *generateLibrary) supportedVariants() []TgtType {
	return []TgtType{m.generateCommon.Properties.Target}
}

func (m *generateLibrary) disable() {
	// This should never actually be called, as we will always support one target
	panic("disable() called on GenerateLibrary")
}

func (m *generateLibrary) setVariant(variant TgtType) {
	// No need to actually track this, as a single target is always supported
}

func (m *generateLibrary) getSplittableProps() *SplittableProps {
	return &m.generateCommon.Properties.FlagArgsBuild.SplittableProps
}

func (m *generateLibrary) FeaturableProperties() []interface{} {
	return append(m.generateCommon.FeaturableProperties(), &m.Properties.GenerateLibraryProps)
}

//// Support singleOutputModule interface

func (m *generateLibrary) outputName() string {
	if m.Properties.Out != nil {
		return *m.Properties.Out
	}
	return m.Name()
}

// Other naming functions, which need to reflect the output name
func (l *generateLibrary) altName() string {
	return l.outputName()
}

func (l *generateLibrary) altShortName() string {
	return l.outputName()
}

//// Support installable

func (m *generateLibrary) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	return m.outputs()
}
