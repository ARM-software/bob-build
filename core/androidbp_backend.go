/*
 * Copyright 2020-2023 Arm Limited.
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
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/fileutils"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
)

var (
	outputFile      = bpwriter.FileFactory()
	buildbpPathsMap = map[string]bool{}
)

type androidBpGenerator struct {
	toolchains toolchain.ToolchainSet
	logger     *warnings.WarningLogger
}

/* Compile time checks for interfaces that must be implemented by androidBpGenerator */
var _ generatorBackend = (*androidBpGenerator)(nil)

// Provides access to the global instance of the Android.bp file writer
func AndroidBpFile() bpwriter.File {
	return outputFile
}

// Translate `bob_alias` to `phony` rules in Android.bp
func (g *androidBpGenerator) aliasActions(a *ModuleAlias, ctx blueprint.ModuleContext) {
	srcs := []string{}

	ctx.WalkDeps(func(child blueprint.Module, parent blueprint.Module) bool {
		// If either the child or parent are not required/enabled, stop
		// traversing. Alias rules are enabled and required by default.
		if !enabledAndRequired(parent) || !enabledAndRequired(child) {
			return false
		}

		switch ctx.OtherModuleType(child) {
		case "bob_resource":
			if r, ok := child.(*ModuleResource); ok {
				r.Properties.GetFiles(ctx).ForEach(func(fp file.Path) bool {
					srcs = append(srcs, r.getAndroidbpResourceName(fp.UnScopedPath()))
					return true
				})
			}
			return false
		case "bob_alias":
			// For alias type, do not add it, instead continue
			// recursing to find the non-alias deps.
			return true
		default:
			name := ctx.OtherModuleName(child)
			if lib, ok := child.(phonyInterface); ok {
				name = lib.shortName()
			}
			srcs = append(srcs, name)
		}
		return false
	})

	// Cannot have phony rules with no deps
	if len(srcs) == 0 {
		return
	}

	mod, err := AndroidBpFile().NewModule("phony", a.Name())
	if err != nil {
		panic(err)
	}

	mod.AddStringList("required", srcs)
}

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

func (g *androidBpGenerator) sourceOutputDir(m blueprint.Module) string {
	return ""
}

func (g *androidBpGenerator) sharedLibsDir(toolchain.TgtType) string {
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

func (g *androidBpGenerator) escapeFlag(s string) string {
	// Soong will handle the escaping of flags, so the androidbp backend
	// just passes them through.
	return s
}

func addProvenanceProps(m bpwriter.Module, props AndroidProps) {
	if props.isProprietary() {
		m.AddString("owner", proptools.String(props.Owner))
		m.AddBool("vendor", true)
		m.AddBool("proprietary", true)
		m.AddBool("soc_specific", true)
	}
}

func addInstallProps(m bpwriter.Module, props *InstallableProps, proprietary bool) {
	installBase, installRel, ok := getSoongInstallPath(props)
	if ok {
		switch installBase {
		case "data":
			m.AddBool("install_in_data", true)
		case "tests":
			/* Eventually we want to install in testcases,
			 * but we can't put binaries there yet:
			 * bpmod.AddBool("install_in_testcases", true)
			 * So place resources in /data/nativetest to align with cc_test.
			 *
			 * `nativetest` has no corresponding `InstallIn...` method,
			 * so request the `/data` partition and add the `nativetest`
			 * part in as another relative component. */
			m.AddBool("install_in_data", true)
			if proprietary {
				// Vendor modules need an additional path element to match cc_test
				installRel = filepath.Join("nativetest", "vendor", installRel)
			} else {
				installRel = filepath.Join("nativetest", installRel)
			}
		default:
			/* Paths like `lib/modules` are implicitly in /system, or /vendor, but
			 * unlike e.g. a library, which would add the `lib` for us, we need to add
			 * it ourselves here - so the whole path is used as the relative part. */
			installRel = filepath.Join(installBase, installRel)
		}
		m.AddString("install_path", installRel)
	}
}

type androidBpSingleton struct {
}

func androidBpSingletonFactory() blueprint.Singleton {
	return &androidBpSingleton{}
}

func collectBuildBpFilesMutator(ctx blueprint.BottomUpMutatorContext) {
	buildbpPathsMap[ctx.BlueprintsFile()] = true
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
		utils.Die("Could not open depfile %s: %v", depfile, err)
	}
	defer file.Close()

	lines := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		utils.Die("Error reading depfile %s: %v", depfile, err)
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
		utils.Die("Could not open %s for hashing: %v", path, err)
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

// A line to match in a specific source file, to identify a particular Soong version.
type codeMatcher struct {
	filename string
	text     string
	result   *bool
}

func (cm *codeMatcher) match() bool {
	if cm.result != nil {
		return *cm.result
	}

	matched := false
	contents, err := ioutil.ReadFile(cm.filename)
	if err == nil {
		matched = strings.Contains(string(contents), cm.text)
	} else {
		fmt.Fprintf(os.Stderr,
			"WARNING: Could not open file %s to determine Soong "+
				"compatibility layer: %s\n"+
				"Compilation of Bob plugins may fail!\n",
			err.Error(), cm.filename)
	}

	cm.result = &matched
	return matched
}

func getSoongCompatFile(config *BobConfig) string {
	type compatVersion struct {
		matches          []codeMatcher
		android_versions []int
		src              string
	}

	listOfAndroidMkEntriesMatcher := codeMatcher{
		filename: "build/soong/android/androidmk.go",
		text:     "\n\tAndroidMkEntries() []AndroidMkEntries\n",
	}

	androidMkExtraEntriesContextMatcher := codeMatcher{
		filename: "build/soong/android/androidmk.go",
		text:     "\ntype AndroidMkExtraEntriesContext interface {\n",
	}

	androidMkSoongInstallTargetsMatcher := codeMatcher{
		filename: "build/soong/android/androidmk.go",
		text:     "a.SetPath(\"LOCAL_SOONG_INSTALLED_MODULE\", base.katiInstalls[len(base.katiInstalls)-1].to)\n",
	}

	// List of compatibility layers, ordered from oldest Soong version
	// support to newest.
	allSoongCompats := []compatVersion{
		// AndroidMkEntries() was made to return an array in 0b0e1b9
		{
			[]codeMatcher{
				listOfAndroidMkEntriesMatcher,
			},
			[]int{9, 10, 11, 12},
			"soong_compat_00_pqr.go",
		},
		// AndroidMkExtraEntriesContext was added in aa25553
		{
			[]codeMatcher{
				listOfAndroidMkEntriesMatcher,
				androidMkExtraEntriesContextMatcher,
			},
			[]int{12},
			"soong_compat_01_AndroidMkExtraEntries_ctx.go",
		},
		// Soong install Mk targets have been added in 6301c3c
		{
			[]codeMatcher{
				listOfAndroidMkEntriesMatcher,
				androidMkExtraEntriesContextMatcher,
				androidMkSoongInstallTargetsMatcher,
			},
			[]int{13},
			"soong_compat_02_AndroidMkSoongInstallTargets.go",
		},
	}

	android_platform_version := config.Properties.GetInt("android_platform_version")

	soongCompats := []compatVersion{}

	// See if we can uniquely identify the required compatibility code based
	// on the Android version
	for _, soongCompat := range allSoongCompats {
		android_versions := soongCompat.android_versions

		for _, v := range android_versions {
			if v == android_platform_version {
				soongCompats = append(soongCompats, soongCompat)
			}
		}
	}

	if len(soongCompats) == 1 {
		return soongCompats[0].src
	} else if len(soongCompats) == 0 {
		fmt.Fprintf(os.Stderr, "WARNING: No available Soong compatibility "+
			"layers found for ANDROID_PLATFORM_VERSION = %d.\n"+
			"Attempting text-based detection.\n",
			android_platform_version)
		soongCompats = allSoongCompats
	}

	// If there are multiple potential options for this Android version, try
	// to differentiate by matching specific lines in Soong's source. Search
	// from newest to oldest - newer Soong versions may contain older code
	// fragments too, so going the other way could mean incorrectly choosing
	// an earlier version.
	for i := len(soongCompats) - 1; i >= 0; i-- {
		allMatchesPassed := true
		for _, m := range soongCompats[i].matches {
			if !m.match() {
				allMatchesPassed = false
				break
			}
		}
		if allMatchesPassed {
			return soongCompats[i].src
		}
	}

	fmt.Fprintf(os.Stderr, "WARNING: Could not find an appropriate Soong "+
		"compatibility layer based on code in build/soong.\n"+
		"WARNING: Falling back to default for this Android version. "+
		"Compilation of Bob plugins may fail!\n")
	return soongCompats[len(soongCompats)-1].src
}

func (s *androidBpSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	sb := &strings.Builder{}

	// read definitions of plugin packages
	pluginTemplate := filepath.Join(getBobDir(), "plugins/Android.bp.in")
	ctx.AddNinjaFileDeps(pluginTemplate)
	content, err := ioutil.ReadFile(pluginTemplate)
	if err != nil {
		utils.Die("%v", err.Error())
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
	text = strings.Replace(text, "@@SOONG_COMPAT@@", getSoongCompatFile(getConfig(ctx)), -1)
	sb.WriteString(text)
	sb.WriteString("\n")

	s.generateBuildbpCheck(ctx, projUid)

	// dump all modules
	AndroidBpFile().Render(sb)

	androidbpFile := getPathInSourceDir("Android.bp")
	err = fileutils.WriteIfChanged(androidbpFile, sb)
	if err != nil {
		utils.Die("%v", err.Error())
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

func (g *androidBpGenerator) getLogger() *warnings.WarningLogger {
	return g.logger
}

func (g *androidBpGenerator) init(ctx *blueprint.Context, config *BobConfig) {
	// Do not run in parallel to avoid locking issues on the map
	ctx.RegisterBottomUpMutator("collect_buildbp", collectBuildBpFilesMutator)
	ctx.RegisterSingletonType("androidbp_singleton", androidBpSingletonFactory)
	g.toolchains.Configure(&config.Properties)
}

func (g *androidBpGenerator) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	return g.toolchains.GetToolchain(tgt)
}
