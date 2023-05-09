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
	"path/filepath"
	"regexp"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// TransformSourceProps contains the properties allowed in the
// bob_transform_source module. This module supports one command execution
// per input file.
type TransformSourceProps struct {
	// The regular expression that is used to transform the source path to the target path.
	Out struct {
		// Regular expression to capture groups from srcs
		Match string
		// Names of outputs, which can use capture groups from match
		Replace []string
		// List of implicit sources. Implicit sources are input files that do not get mentioned on the command line,
		// and are not specified in the explicit sources.
		Implicit_srcs []string
	}

	// Stores the files generated
	ResolvedOut FilePaths `blueprint:"mutated"`
}

func (tsp *TransformSourceProps) inoutForSrc(re *regexp.Regexp, source filePath, depfile *bool, rspfile bool) (io inout) {
	io.in = []string{source.buildPath()}

	for _, rep := range tsp.Out.Replace {
		out := filepath.Join(re.ReplaceAllString(source.localPath(), rep))
		io.out = append(io.out, out)
	}

	if proptools.Bool(depfile) {
		io.depfile = getDepfileName(source.localPath())
	}

	for _, implSrc := range tsp.Out.Implicit_srcs {
		implSrc = re.ReplaceAllString(source.localPath(), implSrc)
		io.implicitSrcs = append(io.implicitSrcs, filepath.Join(source.moduleDir(), implSrc))
	}

	if rspfile {
		io.rspfile = getRspfileName(source.localPath())
	}

	return
}

// The module that can generate sources using a multiple execution
// The command will be run once per src file- with $in being the path in "srcs" and $out being the path transformed
// through the regexp defined by out.match and out.replace. The regular expression that is used is
// in regexp.compiled(out.Match).ReplaceAllString(src[i], out.Replace). See https://golang.org/pkg/regexp/ for more
// information.
// The working directory will be the source directory, and all paths will be relative to the source directory
// if not else noted
type transformSource struct {
	ModuleGenerateCommon
	Properties struct {
		TransformSourceProps
	}
}

// All interfaces supported by filegroup
type transformSourceInterface interface {
	installable
	SourceFileProvider
	SourceFileConsumer
	FileResolver
}

var _ transformSourceInterface = (*transformSource)(nil) // impl check

func (m *transformSource) FeaturableProperties() []interface{} {
	return append(m.ModuleGenerateCommon.FeaturableProperties(), &m.Properties.TransformSourceProps)
}

func (m *transformSource) sourceInfo(ctx blueprint.ModuleContext, g generatorBackend) []filePath {
	return m.GetSrcs(ctx)
}

func (m *transformSource) ResolveFiles(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.getLegacySourceProperties().ResolveFiles(ctx, g)
}

func (m *transformSource) GetSrcs(ctx blueprint.BaseModuleContext) FilePaths {
	return m.getLegacySourceProperties().GetSrcs(ctx)
}

func (m *transformSource) GetDirectSrcs() FilePaths {
	return m.getLegacySourceProperties().GetDirectSrcs()
}

func (m *transformSource) GetSrcTargets() []string {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.Generated_sources
}

func (m *transformSource) OutSrcs() FilePaths {
	return m.Properties.ResolvedOut
}

func (m *transformSource) OutSrcTargets() []string {
	return []string{}
}

func (m *transformSource) ResolveOutSrcs(ctx blueprint.BaseModuleContext) {
	g := getBackend(ctx)
	re := regexp.MustCompile(m.Properties.Out.Match)

	// fmt.Printf("%v:%+v\n", m.Name(), m.GetSrcs(ctx))
	// TODO: Refactor this to share code with generateInouts, right now the ctx type is differnet so no sharing is possible.
	m.GetSrcs(ctx).ForEach(
		func(fp filePath) bool {
			io := m.Properties.inoutForSrc(re, fp, m.ModuleGenerateCommon.Properties.Depfile,
				m.ModuleGenerateCommon.Properties.Rsp_content != nil)
			for _, out := range io.out {
				fp := newGeneratedFilePathFromModule(out, ctx, g)
				m.Properties.ResolvedOut = m.Properties.ResolvedOut.AppendIfUnique(fp)
			}
			return true
		})

}

// Return an inouts structure naming all the files associated with
// each transformSource input.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies
// to out, depfile and rspfile. The output directory (if needed) needs to be
// added in by the backend specific GenerateBuildAction()
func (m *transformSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var inouts []inout
	re := regexp.MustCompile(m.Properties.Out.Match)

	for _, source := range m.sourceInfo(ctx, g) {
		io := m.Properties.inoutForSrc(re, source, m.ModuleGenerateCommon.Properties.Depfile,
			m.ModuleGenerateCommon.Properties.Rsp_content != nil)
		inouts = append(inouts, io)
	}

	return inouts
}

func (m *transformSource) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	// Install everything that we generate
	return m.outputs()
}

func (m *transformSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		g := getBackend(ctx)
		g.transformSourceActions(m, ctx)
	}
}

func (m transformSource) GetProperties() interface{} {
	return m.Properties
}

func transformSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &transformSource{}
	module.ModuleGenerateCommon.init(&config.Properties,
		GenerateProps{}, TransformSourceProps{})

	return module, []interface{}{&module.ModuleGenerateCommon.Properties,
		&module.Properties,
		&module.SimpleName.Properties}
}
