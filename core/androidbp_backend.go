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
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/fileutils"
	"github.com/ARM-software/bob-build/internal/utils"
)

var (
	outputFile      = bpwriter.FileFactory()
	buildbpPathsMap = map[string]bool{}
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
	// On the androidbp backend, sourceDir() is only used for match_src
	// handling and for locating bob scripts. In these cases we want
	// paths relative to ANDROID_BUILD_TOP directory,
	// which is where all commands will be executed from
	return getSourceDir()
}

func (g *androidBpGenerator) bobScriptsDir() string {
	// In the androidbp backend, we just want the relative path to the
	// script directory.
	srcToScripts, _ := filepath.Rel(getSourceDir(), getBobScriptsDir())
	return srcToScripts
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

type androidBpSingleton struct {
}

func androidBpSingletonFactory() blueprint.Singleton {
	return &androidBpSingleton{}
}

func collectBuildBpFilesMutator(mctx blueprint.BottomUpMutatorContext) {
	buildbpPathsMap[mctx.BlueprintsFile()] = true
}

// Extract dependencies from a depfile where:
// * The first line contains the target (and no dependencies)
// * The rest of the file contains dependencies, one file per line
// NOTE: This is not a general-purpose function for parsing depfiles.
// It is only compatible with depfiles satisfying the above (that is,
// with the depfile format used by the config system)
func extractDeps(depfile string) []string {
	file, err := os.Open(depfile)
	if err != nil {
		panic(fmt.Errorf("Could not open depfile %s: %v", depfile, err))
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Errorf("Error reading depfile %s: %v", depfile, err))
	}

	if len(lines) == 0 {
		return []string{}
	}

	deps := lines[1:] // The first line contains the target
	for i, dep := range deps {
		deps[i] = strings.TrimSpace(strings.TrimSuffix(dep, "\\"))
	}

	return deps
}

func hashFileAtPath(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Errorf("Could not open %s for hashing: %v", path, err))
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		panic(err)
	}

	return hash.Sum(nil)
}

// Compute the hash of build.bp and Mconfig files.
func hashBuildConfig(paths []string) string {
	combinedHash := sha1.New()
	for _, path := range paths {
		combinedHash.Write(hashFileAtPath(path))
	}
	return hex.EncodeToString(combinedHash.Sum(nil))
}

func (s *androidBpSingleton) generateBuildbpCheck(ctx blueprint.SingletonContext, projUid string) {
	g := getConfig(ctx).Generator

	bpmod, err := AndroidBpFile().NewModule("genrule", "_check_buildbp_updates_"+projUid)
	if err != nil {
		panic(err)
	}

	checkerScript := getBackendPathInBobScriptsDir(g, "verify_hash.py")

	buildbpPathsList := utils.SortedKeysBoolMap(buildbpPathsMap)
	prefixedBuildbpPathsList := utils.PrefixDirs(buildbpPathsList, getSourceDir())

	configDeps := []string{}
	prefixedConfigDeps := extractDeps(configFile + ".d")
	for _, path := range prefixedConfigDeps {
		relPath, err := filepath.Rel(getSourceDir(), path)
		if err != nil {
			panic(err)
		}
		configDeps = append(configDeps, relPath)
	}

	srcs := append(buildbpPathsList, configDeps...)
	prefixedSrcs := append(prefixedBuildbpPathsList, prefixedConfigDeps...)

	hash := hashBuildConfig(prefixedSrcs)

	ctx.AddNinjaFileDeps(configFile + ".d")

	bpmod.AddStringList("srcs", srcs)
	bpmod.AddStringList("out", []string{"androidbp_up_to_date"})
	bpmod.AddStringList("tool_files", []string{checkerScript})
	bpmod.AddStringCmd("cmd",
		[]string{
			"python", "$(location " + checkerScript + ")",
			"--hash", hash,
			"--out", "$(out)",
			"--", "$(in)",
		})
}

func (s *androidBpSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	sb := &strings.Builder{}

	// read definitions of plugin packages
	content, err := ioutil.ReadFile(filepath.Join(getBobDir(), "plugins/Android.bp.in"))
	if err != nil {
		utils.Exit(1, err.Error())
	}

	// use source dir to get project-unique identifier,
	// generate 10 chars in hex based on its hash
	h := sha1.New()
	h.Write([]byte(getSourceDir()))
	projUid := hex.EncodeToString((h.Sum(nil)[:5]))
	// bob dir must be relative to source dir
	srcToBobDir, _ := filepath.Rel(getSourceDir(), getBobDir())

	// substitute template variables
	text := string(content)
	text = strings.Replace(text, "@@PROJ_UID@@", projUid, -1)
	text = strings.Replace(text, "@@BOB_DIR@@", srcToBobDir, -1)
	sb.WriteString(text)
	sb.WriteString("\n")

	s.generateBuildbpCheck(ctx, projUid)

	// dump all modules
	AndroidBpFile().Render(sb)

	androidbpFile := getPathInSourceDir("Android.bp")
	err = fileutils.WriteIfChanged(androidbpFile, sb)
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
	// Do not run in parallel to avoid locking issues on the map
	ctx.RegisterBottomUpMutator("collect_buildbp", collectBuildBpFilesMutator)

	ctx.RegisterSingletonType("androidbp_singleton", androidBpSingletonFactory)

	g.toolchainSet.parseConfig(config)
}
