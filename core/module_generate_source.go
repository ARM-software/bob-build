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
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

// GenerateSourceProps are properties of 'bob_generate_source', i.e. a module
// type which can generate sources using a single execution
// The command will be run once - with $in being the paths in "srcs" and $out being the paths in "out".
// The working directory will be the source directory, and all paths will be relative to the source directory
// if not else noted.
type GenerateSourceProps struct {
	// The list of files that will be output.
	Out []string
	// List of implicit sources. Implicit sources are input files that do not get
	// mentioned on the command line, and are not specified in the explicit sources.
	Implicit_srcs []string
	// Implicit source files that should not be included. Use with care.
	Exclude_implicit_srcs []string

	ResolvedOut       FilePaths `blueprint:"mutated"`
	ResolvedImplicits FilePaths `blueprint:"mutated"`
}

type generateSource struct {
	generateCommon
	Properties struct {
		GenerateSourceProps
	}
}

type generateSourceInterface interface {
	installable
	pathProcessor
	FileResolver
	SourceFileProvider
	SourceFileConsumer
}

var _ generateSourceInterface = (*generateSource)(nil) // impl check

func (m *generateSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.generateSourceActions(m, ctx)
	}
}

func (m *generateSource) FeaturableProperties() []interface{} {
	return append(m.generateCommon.FeaturableProperties(), &m.Properties.GenerateSourceProps)
}

func (m *generateSource) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.Implicit_srcs = utils.PrefixDirs(m.Properties.Implicit_srcs, projectModuleDir(ctx))
	m.Properties.Exclude_implicit_srcs = utils.PrefixDirs(m.Properties.Exclude_implicit_srcs, projectModuleDir(ctx))
	m.generateCommon.processPaths(ctx, g)
}

func (m *generateSource) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	// Resolve sources.
	gc, _ := getGenerateCommon(m)
	gc.Properties.LegacySourceProps.ResolveFiles(ctx, g)

	// Resolve output files
	outs := FilePaths{}
	for _, out := range m.Properties.Out {
		fp := newGeneratedFilePathFromModule(out, ctx, g)
		outs = outs.AppendIfUnique(fp)
	}

	implicits := FilePaths{}
	for _, implicit := range glob(ctx, m.Properties.Implicit_srcs, m.Properties.Exclude_implicit_srcs) {
		fp := newSourceFilePath(implicit, ctx, g)
		implicits = implicits.AppendIfUnique(fp)
	}

	m.Properties.ResolvedOut = outs
	m.Properties.ResolvedImplicits = implicits

}

func (m *generateSource) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.LegacySourceProps.GetSrcs(ctx)
}

func (m *generateSource) GetDirectSrcs() FilePaths {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.LegacySourceProps.GetDirectSrcs()
}

func (m *generateSource) GetImplicits(ctx blueprint.BaseModuleContext) FilePaths {
	return m.Properties.ResolvedImplicits
}

func (m *generateSource) GetSrcTargets() []string {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.Generated_sources
}

func (m *generateSource) OutSrcs() FilePaths {
	return m.Properties.ResolvedOut
}

func (m *generateSource) OutSrcTargets() []string {
	return []string{}
}

// Return an inouts structure naming all the files associated with a
// generateSource's inputs.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies
// to out, depfile and rspfile. The output directory (if needed) needs to be
// added in by the backend specific GenerateBuildAction()
func (m *generateSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
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

	io.out = m.Properties.Out
	if depfile, ok := m.getDepfile(); ok {
		io.depfile = depfile
	}
	if rspfile, ok := m.getRspfile(); ok {
		io.rspfile = rspfile
	}

	return []inout{io}
}

func (m *generateSource) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	// Install everything that we generate
	return m.outputs()
}

func (m generateSource) GetProperties() interface{} {
	return m.Properties
}

func generateSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateSource{}
	module.generateCommon.init(&config.Properties,
		GenerateProps{}, GenerateSourceProps{})

	return module, []interface{}{&module.generateCommon.Properties, &module.Properties,
		&module.SimpleName.Properties}
}
