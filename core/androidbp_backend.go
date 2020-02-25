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

func (g *androidBpGenerator) aliasActions(*alias, blueprint.ModuleContext) {}

func (g *androidBpGenerator) buildDir() string {
	// The androidbp backend writes an Android.bp file, which should
	// never reference an actual output directory (which will be
	// chosen by Soong). Therefore this function returns an empty
	// string.
	return ""
}

func (g *androidBpGenerator) sourceDir() string {
	// The androidbp backend writes paths into an Android.bp file in
	// the project directory. All paths should be relative to that
	// file, so there should be no need for the source directory.
	return ""
}

func (g *androidBpGenerator) bobScriptsDir() string {
	// In the androidbp backend, we just want the relative path to the
	// script directory.
	srcToScripts, _ := filepath.Rel(getSourceDir(), getBobScriptsDir())
	return filepath.Join(g.sourceDir(), srcToScripts)
}

func (g *androidBpGenerator) sharedLibsDir(tgtType) string {
	// When writing link commands, it's common to put all the shared
	// libraries in a single location to make it easy for the linker to
	// find them. This function tells us where this is for the current
	// generatorBackend.
	//
	// In the androidbp backend, we don't write link command lines. Soong
	// will do this after processing the generated Android.bp. Therefore
	// this function just returns an empty string.
	return ""
}

//// Module specific functions identifying where backends expect module output to go.

// The androidbp backend writes Android.bp files, which should never
// need to reference files in their actual output location. Soong will
// add the necessary paths when it runs. Therefore all these return an
// empty string.
func (g *androidBpGenerator) sourceOutputDir(*generateCommon) string   { return "" }
func (g *androidBpGenerator) binaryOutputDir(*binary) string           { return "" }
func (g *androidBpGenerator) staticLibOutputDir(*staticLibrary) string { return "" }
func (g *androidBpGenerator) sharedLibOutputDir(*sharedLibrary) string { return "" }
func (g *androidBpGenerator) kernelModOutputDir(*kernelModule) string  { return "" }

//// End module specific functions

type androidBpSingleton struct {
}

func androidBpSingletonFactory() blueprint.Singleton {
	return &androidBpSingleton{}
}

func (s *androidBpSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	sb := &strings.Builder{}
	AndroidBpFile().Render(sb)

	androidbpFile := getPathInSourceDir("Android.bp")
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
