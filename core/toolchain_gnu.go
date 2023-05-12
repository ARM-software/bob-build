/*
 * Copyright 2018-2020, 2023 Arm Limited.
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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
)

type toolchainGnu interface {
	toolchain
	getBinDirs() []string
	getStdCxxHeaderDirs() []string
	getInstallDir() string
}

type toolchainGnuCommon struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	objdumpBinary string
	gccBinary     string
	gxxBinary     string
	linker        linker
	prefix        string
	cflags        []string // Flags for both C and C++
	ldflags       []string // Linker flags, including anything required for C++
	binDir        string
	flagCache     *flagSupportedCache
}

type toolchainGnuNative struct {
	toolchainGnuCommon
}

type toolchainGnuCross struct {
	toolchainGnuCommon
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

func (tc toolchainGnuCommon) getLinker() linker {
	return tc.linker
}

func (tc toolchainGnuCommon) getStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainGnuCommon) getLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainGnuCommon) getBinDirs() []string {
	return []string{tc.binDir}
}

func (tc toolchainGnuCommon) checkFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

// The libstdc++ headers shipped with GCC toolchains are stored, relative to
// the `prefix-gcc` binary's location, in `../$ARCH/include/c++/$VERSION` and
// `../$ARCH/include/c++/$VERSION/$ARCH`. This function returns $ARCH. This is
// generally the same as the compiler prefix, but because the prefix can
// contain the path to the compiler as well, we instead obtain it by trying the
// `-print-multiarch` and `-dumpmachine` options.
func (tc toolchainGnuCommon) getTargetTripleHeaderSubdir() string {
	ccBinary, flags := tc.getCCompiler()
	cmd := exec.Command(ccBinary, utils.NewStringSlice(flags, []string{"-print-multiarch"})...)
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
	cmd = exec.Command(ccBinary, utils.NewStringSlice(flags, []string{"-dumpmachine"})...)
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

func (tc toolchainGnuCommon) getInstallDir() string {
	return filepath.Dir(tc.binDir)
}

// Prefixed standalone toolchains (e.g. aarch64-linux-gnu-gcc) often ship with a
// directory of symlinks containing un-prefixed names e.g. just 'ld', instead of
// 'aarch64-linux-gnu-ld'. Some Clang installations won't use the prefix, even
// when passed the --gcc-toolchain option, so add the unprefixed version to the
// binary search path.
func (tc toolchainGnuCross) getBinDirs() []string {
	dirs := tc.toolchainGnuCommon.getBinDirs()
	triple := tc.getTargetTripleHeaderSubdir()

	unprefixedBinDir := filepath.Join(tc.getInstallDir(), triple, "bin")
	if fi, err := os.Stat(unprefixedBinDir); !os.IsNotExist(err) && fi.IsDir() {
		dirs = append(dirs, unprefixedBinDir)
	}
	return dirs
}

func (tc toolchainGnuNative) getStdCxxHeaderDirs() []string {
	installDir := tc.getInstallDir()
	triple := tc.getTargetTripleHeaderSubdir()
	return []string{
		filepath.Join(installDir, "include", "c++", tc.getVersion()),
		filepath.Join(installDir, "include", "c++", tc.getVersion(), triple),
	}
}

func (tc toolchainGnuCross) getStdCxxHeaderDirs() []string {
	installDir := tc.getInstallDir()
	triple := tc.getTargetTripleHeaderSubdir()
	return []string{
		filepath.Join(installDir, triple, "include", "c++", tc.getVersion()),
		filepath.Join(installDir, triple, "include", "c++", tc.getVersion(), triple),
	}
}

func newToolchainGnuCommon(config *BobConfig, tgt TgtType) (tc toolchainGnuCommon) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_gnu_prefix")
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")

	tc.gccBinary = tc.prefix + props.GetString(string(tgt)+"_gnu_cc_binary")
	tc.gxxBinary = tc.prefix + props.GetString(string(tgt)+"_gnu_cxx_binary")
	tc.binDir = filepath.Dir(getToolPath(tc.gccBinary))

	sysroot := props.GetString(string(tgt) + "_sysroot")
	if sysroot != "" {
		tc.cflags = append(tc.cflags, "--sysroot="+sysroot)
		tc.ldflags = append(tc.ldflags, "--sysroot="+sysroot)
	}

	flags := strings.Split(config.Properties.GetString(string(tgt)+"_gnu_flags"), " ")
	tc.cflags = append(tc.cflags, flags...)
	tc.ldflags = append(tc.ldflags, flags...)

	tc.linker = newDefaultLinker(tc.gxxBinary, tc.ldflags, []string{})
	tc.flagCache = newFlagCache()

	return
}

func newToolchainGnuNative(config *BobConfig) (tc toolchainGnuNative) {
	tc.toolchainGnuCommon = newToolchainGnuCommon(config, TgtTypeHost)
	return
}

func newToolchainGnuCross(config *BobConfig) (tc toolchainGnuCross) {
	tc.toolchainGnuCommon = newToolchainGnuCommon(config, TgtTypeTarget)
	return
}
