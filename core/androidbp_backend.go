/*
 * Copyright 2020 Arm Limited.
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
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/fileutils"
	"github.com/ARM-software/bob-build/internal/utils"
)

var (
	outputFile = bpwriter.FileFactory()
)

type androidBpGenerator struct {
	toolchainSet
}

/* Compile time checks for interfaces that must be implemented by androidBpGenerator */
var _ generatorBackend = (*androidBpGenerator)(nil)

// Provides access to the global instance of the Android.bp file writer
func AndroidBpFile() bpwriter.File {
	return outputFile
}

func (g *androidBpGenerator) aliasActions(*alias, blueprint.ModuleContext)               {}
func (g *androidBpGenerator) kernelModuleActions(*kernelModule, blueprint.ModuleContext) {}
func (g *androidBpGenerator) resourceActions(*resource, blueprint.ModuleContext)         {}

func (g *androidBpGenerator) generateSourceActions(*generateSource, blueprint.ModuleContext, []inout) {
}
func (g *androidBpGenerator) genBinaryActions(*generateBinary, blueprint.ModuleContext, []inout) {}
func (g *androidBpGenerator) genSharedActions(*generateSharedLibrary, blueprint.ModuleContext, []inout) {
}
func (g *androidBpGenerator) genStaticActions(*generateStaticLibrary, blueprint.ModuleContext, []inout) {
}
func (g *androidBpGenerator) transformSourceActions(*transformSource, blueprint.ModuleContext, []inout) {
}

func (g *androidBpGenerator) buildDir() string                         { return "" }
func (g *androidBpGenerator) sourcePrefix() string                     { return "" }
func (g *androidBpGenerator) sharedLibsDir(tgtType) string             { return "" }
func (g *androidBpGenerator) sourceOutputDir(*generateCommon) string   { return "" }
func (g *androidBpGenerator) binaryOutputDir(*binary) string           { return "" }
func (g *androidBpGenerator) staticLibOutputDir(*staticLibrary) string { return "" }
func (g *androidBpGenerator) sharedLibOutputDir(*sharedLibrary) string { return "" }
func (g *androidBpGenerator) kernelModOutputDir(*kernelModule) string  { return "" }

type androidBpSingleton struct {
}

func androidBpSingletonFactory() blueprint.Singleton {
	return &androidBpSingleton{}
}

func (s *androidBpSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	sb := &strings.Builder{}
	AndroidBpFile().Render(sb)

	androidbpFile := filepath.Join(srcdir, "Android.bp")
	err := fileutils.WriteIfChanged(androidbpFile, sb)
	if err != nil {
		utils.Exit(1, err.Error())
	}

	// Blueprint does not output package context dependencies unless
	// the package context outputs a variable, pool or rule to the
	// build.ninja.
	//
	// The Android.bp backend does not create variables, pools or
	// rules since the build logic is actually written in Android.bp files.
	// Therefore write a dummy ninja target to ensure that the bob
	// package context dependencies are output.
	//
	// We make the target optional, so that it doesn't execute when
	// ninja runs without a target.
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     dummyRule,
			Outputs:  []string{androidbpFile},
			Optional: true,
		})
}

func (g *androidBpGenerator) init(ctx *blueprint.Context, config *bobConfig) {
	ctx.RegisterSingletonType("androidbp_singleton", androidBpSingletonFactory)

	g.toolchainSet.parseConfig(config)
}
