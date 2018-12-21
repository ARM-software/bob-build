/*
 * Copyright 2018 Arm Limited.
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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/utils"
)

type toolchain interface {
	getAssembler() (tool string, flags []string)
	getCCompiler() (tool string, flags []string)
	getCXXCompiler() (tool string, flags []string)
	getArchiver() (tool string, flags []string)
}

func getToolchainBinaryPath(tc toolchain) string {
	ccBinary, _ := tc.getCCompiler()
	if !filepath.IsAbs(ccBinary) {
		path, err := exec.LookPath(ccBinary)
		if err != nil {
			panic(err)
		}
		ccBinary = path
	}
	return filepath.Dir(ccBinary)
}

func getToolchainInstallDir(tc toolchain) string {
	return filepath.Dir(getToolchainBinaryPath(tc))
}

type toolchainGnu interface {
	toolchain
	getBinDirs() []string
	getStdCxxHeaderDirs() []string
}

type toolchainGnuCommon struct {
	arBinary  string
	asBinary  string
	gccBinary string
	gxxBinary string
	cflags    []string // Flags for both C and C++
}

type toolchainGnuNative struct {
	toolchainGnuCommon
}

type toolchainGnuCross struct {
	toolchainGnuCommon
	prefix string
}

func (tc toolchainGnuCommon) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainGnuCommon) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainGnuCommon) getCCompiler() (string, []string) {
	return tc.gccBinary, tc.cflags
}

func (tc toolchainGnuCommon) getCXXCompiler() (tool string, flags []string) {
	return tc.gxxBinary, tc.cflags
}

func (tc toolchainGnuCommon) getBinDirs() []string {
	return []string{getToolchainBinaryPath(tc)}
}

// The libstdc++ headers shipped with GCC toolchains are stored, relative to
// the `prefix-gcc` binary's location, in `../$ARCH/include/c++/$VERSION` and
// `../$ARCH/include/c++/$VERSION/$ARCH`. This function returns $ARCH. This is
// generally the same as the compiler prefix, but because the prefix can
// contain the path to the compiler as well, we instead obtain it by trying the
// `-print-multiarch` and `-dumpmachine` options.
func (tc toolchainGnuCommon) getTargetTripleHeaderSubdir() string {
	ccBinary, flags := tc.getCCompiler()
	cmd := exec.Command(ccBinary, append(flags, "-print-multiarch")...)
	bytes, err := cmd.Output()
	if err == nil {
		target := strings.TrimSpace(string(bytes))
		if len(target) > 0 {
			return target
		}
	}

	// Some toolchains will output nothing for -print-multiarch, so try
	// -dumpmachine if it didn't work (-dumpmachine works for most
	// toolchains, but will ignore options like '-m32').
	cmd = exec.Command(ccBinary, append(flags, "-dumpmachine")...)
	bytes, err = cmd.Output()
	if err != nil {
		panic(fmt.Errorf("Couldn't get arch directory for compiler %s: %v", ccBinary, err))
	}
	return strings.TrimSpace(string(bytes))
}

func (tc toolchainGnuCommon) getVersion() string {
	ccBinary, _ := tc.getCCompiler()
	cmd := exec.Command(ccBinary, "-dumpversion")
	bytes, err := cmd.Output()
	if err != nil {
		panic(fmt.Errorf("Couldn't get version for compiler %s: %v", ccBinary, err))
	}
	return strings.TrimSpace(string(bytes))
}

type toolchainArm struct {
	arBinary  string
	asBinary  string
	ccBinary  string
	cxxBinary string
	cflags    []string // Flags for both C and C++
}

func (tc toolchainArm) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainArm) getCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainArm) getCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

func (tc toolchainArm) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

// Prefixed standalone toolchains (e.g. aarch64-linux-gnu-gcc) often ship with a
// directory of symlinks containing un-prefixed names e.g. just 'ld', instead of
// 'aarch64-linux-gnu-ld'. Some Clang installations won't use the prefix, even
// when passed the --gcc-toolchain option, so add the unprefixed version to the
// binary search path.
func (tc toolchainGnuCross) getBinDirs() []string {
	dirs := tc.toolchainGnuCommon.getBinDirs()

	target := strings.TrimSuffix(tc.prefix, "-")

	unprefixedBinDir := filepath.Join(getToolchainInstallDir(tc), target, "bin")
	if fi, err := os.Stat(unprefixedBinDir); !os.IsNotExist(err) && fi.IsDir() {
		dirs = append(dirs, unprefixedBinDir)
	}

	return dirs
}

func (tc toolchainGnuNative) getStdCxxHeaderDirs() []string {
	installDir := getToolchainInstallDir(tc)
	triple := tc.getTargetTripleHeaderSubdir()
	return []string{
		filepath.Join(installDir, "include", "c++", tc.getVersion()),
		filepath.Join(installDir, "include", "c++", tc.getVersion(), triple),
	}
}

func (tc toolchainGnuCross) getStdCxxHeaderDirs() []string {
	installDir := getToolchainInstallDir(tc)
	triple := tc.getTargetTripleHeaderSubdir()
	return []string{
		filepath.Join(installDir, triple, "include", "c++", tc.getVersion()),
		filepath.Join(installDir, triple, "include", "c++", tc.getVersion(), triple),
	}
}

func newToolchainGnuNative(config *bobConfig) (tc toolchainGnuNative) {
	props := config.Properties
	tc.arBinary = props.GetString("ar_binary")
	tc.asBinary = props.GetString("as_binary")
	tc.gccBinary = props.GetString("gnu_cc_binary")
	tc.gxxBinary = props.GetString("gnu_cxx_binary")
	return
}

func newToolchainGnuCross(config *bobConfig) (tc toolchainGnuCross) {
	props := config.Properties
	tc.prefix = props.GetString("target_gnu_toolchain_prefix")
	tc.arBinary = tc.prefix + props.GetString("ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")
	tc.gccBinary = tc.prefix + props.GetString("gnu_cc_binary")
	tc.gxxBinary = tc.prefix + props.GetString("gnu_cxx_binary")
	tc.cflags = strings.Split(props.GetString("target_gnu_flags"), " ")
	return
}

func newToolchainArmCross(config *bobConfig) (tc toolchainArm) {
	props := config.Properties
	tc.arBinary = props.GetString("arm_ar_binary")
	tc.asBinary = props.GetString("arm_as_binary")
	tc.ccBinary = props.GetString("arm_cc_binary")
	tc.cxxBinary = props.GetString("arm_cxx_binary")
	tc.cflags = strings.Split(props.GetString("target_arm_flags"), " ")
	return
}

func newToolchainArmNative(config *bobConfig) (tc toolchainArm) {
	props := config.Properties
	tc.arBinary = props.GetString("arm_ar_binary")
	tc.asBinary = props.GetString("arm_as_binary")
	tc.ccBinary = props.GetString("arm_cc_binary")
	tc.cxxBinary = props.GetString("arm_cxx_binary")
	return
}

type toolchainClangCommon struct {
	// Options read from the config:
	clangBinary   string
	clangxxBinary string

	// Use the GNU toolchain's 'ar' and 'as', as well as its libstdc++
	// headers if required
	gnu toolchainGnu

	// Calculated during toolchain initialization:
	cflags   []string // Flags for both C and C++
	cxxflags []string // Flags just for C++
}

type toolchainClangNative struct {
	toolchainClangCommon
}

type toolchainClangCross struct {
	toolchainClangCommon
	target string
}

func (tc toolchainClangCommon) getArchiver() (string, []string) {
	return tc.gnu.getArchiver()
}

func (tc toolchainClangCommon) getAssembler() (string, []string) {
	return tc.gnu.getAssembler()
}

func (tc toolchainClangCommon) getCCompiler() (string, []string) {
	return tc.clangBinary, tc.cflags
}

func (tc toolchainClangCommon) getCXXCompiler() (string, []string) {
	return tc.clangxxBinary, tc.cxxflags
}

func newToolchainClangCommon(config *bobConfig, gnu toolchainGnu) (tc toolchainClangCommon) {
	props := config.Properties
	tc.clangBinary = props.GetString("clang_cc_binary")
	tc.clangxxBinary = props.GetString("clang_cxx_binary")
	tc.gnu = gnu

	// Tell Clang where the GNU toolchain is installed, so it can use its
	// headers and libraries, for example, if we are using libstdc++.
	tc.cflags = append(tc.cflags, "--gcc-toolchain="+getToolchainInstallDir(tc.gnu))

	// Add the GNU toolchain's binary directories to Clang's binary search
	// path, so that Clang can find the correct linker. If the GNU toolchain
	// is a "system" toolchain (e.g. in /usr/bin), its binaries will already
	// be in Clang's search path, so these arguments have no effect.
	tc.cflags = append(tc.cflags, utils.PrefixAll(tc.gnu.getBinDirs(), "-B")...)

	return
}

func newToolchainClangNative(config *bobConfig) (tc toolchainClangNative) {
	gnu := newToolchainGnuNative(config)
	tc.toolchainClangCommon = newToolchainClangCommon(config, gnu)

	// Combine cflags and cxxflags once here, to avoid appending during
	// every call to getCXXCompiler().
	tc.cxxflags = append(tc.cxxflags, tc.cflags...)

	return
}

func newToolchainClangCross(config *bobConfig) (tc toolchainClangCross) {
	gnu := newToolchainGnuCross(config)
	tc.toolchainClangCommon = newToolchainClangCommon(config, gnu)

	props := config.Properties
	tc.target = props.GetString("target_clang_triple")
	sysroot := props.GetString("target_sysroot")

	if sysroot != "" {
		if tc.target == "" {
			panic(errors.New("TARGET_CLANG_TRIPLE is not set"))
		}
		tc.cflags = append(tc.cflags, "--sysroot", sysroot)

		tc.cxxflags = append(tc.cxxflags,
			utils.PrefixAll(tc.gnu.getStdCxxHeaderDirs(), "-isystem ")...)
	}
	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
	}

	// Combine cflags and cxxflags once here, to avoid appending during
	// every call to getCXXCompiler().
	tc.cxxflags = append(tc.cxxflags, tc.cflags...)

	return
}

type toolchainSet struct {
	host   toolchain
	target toolchain
}

func (tcs *toolchainSet) getToolchain(tgtType string) toolchain {
	if tgtType == tgtTypeHost {
		return tcs.host
	}
	return tcs.target
}

func (tcs *toolchainSet) parseConfig(config *bobConfig) {
	props := config.Properties

	if props.GetBool("target_toolchain_clang") {
		tcs.target = newToolchainClangCross(config)
	} else if props.GetBool("target_toolchain_gnu") {
		tcs.target = newToolchainGnuCross(config)
	} else if props.GetBool("target_toolchain_arm") {
		tcs.target = newToolchainArmCross(config)
	} else {
		panic(errors.New("no usable target compiler toolchain configured"))
	}

	if props.GetBool("host_toolchain_clang") {
		tcs.host = newToolchainClangNative(config)
	} else if props.GetBool("host_toolchain_gnu") {
		tcs.host = newToolchainGnuNative(config)
	} else if props.GetBool("host_toolchain_arm") {
		tcs.host = newToolchainArmNative(config)
	} else {
		panic(errors.New("no usable host compiler toolchain configured"))
	}
}
