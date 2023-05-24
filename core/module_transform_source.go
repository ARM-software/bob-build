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

	"github.com/ARM-software/bob-build/core/file"
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
	ResolvedOut file.Paths `blueprint:"mutated"`
}

func (tsp *TransformSourceProps) inoutForSrc(re *regexp.Regexp, source file.Path, depfile *bool, rspfile bool) (io inout) {
	io.in = []string{source.BuildPath()}

	for _, rep := range tsp.Out.Replace {
		// TODO: figure out the outs here.
		out := filepath.Join(re.ReplaceAllString(source.ScopedPath(), rep))
		io.out = append(io.out, out)
	}

	if proptools.Bool(depfile) {
		io.depfile = getDepfileName(source.UnScopedPath())
	}

	for _, implSrc := range tsp.Out.Implicit_srcs {
		implSrc = re.ReplaceAllString(source.UnScopedPath(), implSrc)
		io.implicitSrcs = append(io.implicitSrcs, source.BuildPath())
	}

	if rspfile {
		io.rspfile = getRspfileName(source.UnScopedPath())
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
type ModuleTransformSource struct {
	ModuleGenerateCommon
	Properties struct {
		TransformSourceProps
	}
}

// All interfaces supported by filegroup
type transformSourceInterface interface {
	installable
	DynamicFileProvider
	FileConsumer
	FileResolver
}

var _ transformSourceInterface = (*ModuleTransformSource)(nil) // impl check

func (m *ModuleTransformSource) FeaturableProperties() []interface{} {
	return append(m.ModuleGenerateCommon.FeaturableProperties(), &m.Properties.TransformSourceProps)
}

func (m *ModuleTransformSource) sourceInfo(ctx blueprint.ModuleContext, g generatorBackend) []file.Path {
	return m.GetFiles(ctx)
}

func (m *ModuleTransformSource) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.getLegacySourceProperties().ResolveFiles(ctx)
}

func (m *ModuleTransformSource) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.getLegacySourceProperties().GetFiles(ctx)
}

func (m *ModuleTransformSource) GetDirectFiles() file.Paths {
	return m.getLegacySourceProperties().GetDirectFiles()
}

func (m *ModuleTransformSource) GetTargets() []string {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.Generated_sources
}

func (m *ModuleTransformSource) OutFiles(g generatorBackend) file.Paths {
	return m.Properties.ResolvedOut
}

func (m *ModuleTransformSource) OutFileTargets() []string {
	return []string{}
}

func (m *ModuleTransformSource) ResolveOutFiles(ctx blueprint.BaseModuleContext) {
	re := regexp.MustCompile(m.Properties.Out.Match)

	// TODO: Refactor this to share code with generateInouts, right now the ctx type is different so no sharing is possible.

	m.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			io := m.Properties.inoutForSrc(re, fp, m.ModuleGenerateCommon.Properties.Depfile,
				m.ModuleGenerateCommon.Properties.Rsp_content != nil)
			for _, out := range io.out {
				fp := file.NewPath(out, ctx.ModuleName(), file.TypeGenerated)
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
func (m *ModuleTransformSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var inouts []inout
	re := regexp.MustCompile(m.Properties.Out.Match)

	for _, source := range m.sourceInfo(ctx, g) {
		io := m.Properties.inoutForSrc(re, source, m.ModuleGenerateCommon.Properties.Depfile,
			m.ModuleGenerateCommon.Properties.Rsp_content != nil)
		inouts = append(inouts, io)
	}

	return inouts
}

func (m *ModuleTransformSource) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	// Install everything that we generate
	return m.outputs()
}

func (m *ModuleTransformSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).transformSourceActions(m, ctx)
	}
}

func (m ModuleTransformSource) GetProperties() interface{} {
	return m.Properties
}

func transformSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleTransformSource{}
	module.ModuleGenerateCommon.init(&config.Properties,
		GenerateProps{}, TransformSourceProps{})

	return module, []interface{}{&module.ModuleGenerateCommon.Properties,
		&module.Properties,
		&module.SimpleName.Properties}
}
