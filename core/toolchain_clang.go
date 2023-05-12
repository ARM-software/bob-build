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
	"github.com/ARM-software/bob-build/internal/utils"
)

type toolchainClangCommon struct {
	// Options read from the config:
	arBinary       string
	asBinary       string
	objcopyBinary  string
	objdumpBinary  string
	clangBinary    string
	clangxxBinary  string
	linker         linker
	prefix         string
	useGnuBinutils bool

	// Use the GNU toolchain's 'ar' and 'as', as well as its libstdc++
	// headers if required
	gnu toolchainGnu

	// Calculated during toolchain initialization:
	cflags   []string // Flags for both C and C++
	cxxflags []string // Flags just for C++
	ldflags  []string // Linker flags, including anything required for C++
	ldlibs   []string // Linker libraries

	target    string
	flagCache *flagSupportedCache
}

type toolchainClangNative struct {
	toolchainClangCommon
}

type toolchainClangCross struct {
	toolchainClangCommon
}

func (tc toolchainClangCommon) getArchiver() (string, []string) {
	if tc.useGnuBinutils {
		return tc.gnu.getArchiver()
	}
	return tc.arBinary, []string{}
}

func (tc toolchainClangCommon) getAssembler() (string, []string) {
	if tc.useGnuBinutils {
		return tc.gnu.getAssembler()
	}
	return tc.asBinary, []string{}
}

func (tc toolchainClangCommon) getCCompiler() (string, []string) {
	return tc.clangBinary, tc.cflags
}

func (tc toolchainClangCommon) getCXXCompiler() (string, []string) {
	return tc.clangxxBinary, tc.cxxflags
}

func (tc toolchainClangCommon) getLinker() linker {
	return newDefaultLinker(tc.clangxxBinary, tc.ldflags, tc.ldlibs)
}

func (tc toolchainClangCommon) getStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainClangCommon) getLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainClangCommon) checkFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func newToolchainClangCommon(config *BobConfig, tgt TgtType) (tc toolchainClangCommon) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_clang_prefix")

	// This assumes arBinary and asBinary are either in the path, or the same directory as clang.
	// This is not necessarily the case. This will need to be updated when we support clang on linux without a GNU toolchain.
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")

	tc.clangBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cc_binary")
	tc.clangxxBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cxx_binary")

	tc.target = props.GetString(string(tgt) + "_clang_triple")

	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
		tc.ldflags = append(tc.ldflags, "-target", tc.target)
	}

	sysroot := props.GetString(string(tgt) + "_sysroot")
	if sysroot != "" {
		tc.cflags = append(tc.cflags, "--sysroot="+sysroot)
		tc.ldflags = append(tc.ldflags, "--sysroot="+sysroot)
	}

	stl := props.GetString(string(tgt) + "_clang_stl_library")
	rt := props.GetString(string(tgt) + "_clang_compiler_runtime")
	useGnuCrt := props.GetBool(string(tgt) + "_clang_use_gnu_crt")
	useGnuStl := props.GetBool(string(tgt) + "_clang_use_gnu_stl")
	useGnuLibgcc := props.GetBool(string(tgt) + "_clang_use_gnu_libgcc")

	tc.useGnuBinutils = props.GetBool(string(tgt) + "_clang_use_gnu_binutils")

	if tc.useGnuBinutils || useGnuStl || useGnuCrt || useGnuLibgcc {
		if tgt == TgtTypeHost {
			tc.gnu = newToolchainGnuNative(config)
		} else {
			tc.gnu = newToolchainGnuCross(config)
		}
	}

	if stl != "" {
		tc.cxxflags = append(tc.cxxflags, "--stdlib=lib"+stl)
		tc.ldflags = append(tc.ldflags, "--stdlib=lib"+stl)
	}

	if rt != "" {
		tc.cflags = append(tc.cflags, "--rtlib="+rt)
		tc.ldflags = append(tc.ldflags, "--rtlib="+rt)
	}

	binDirs := []string{}

	if useGnuCrt || useGnuLibgcc || useGnuStl {
		// Tell Clang where the GNU toolchain is installed, so it can use its
		// headers and libraries, for example, if we are using libstdc++.
		gnuInstallArg := "--gcc-toolchain=" + tc.gnu.getInstallDir()
		tc.cflags = append(tc.cflags, gnuInstallArg)
		tc.ldflags = append(tc.ldflags, gnuInstallArg)
	}
	if useGnuCrt {
		binDirs = append(binDirs, getFileNameDir(tc.gnu, "crt1.o")...)
	}
	if tc.useGnuBinutils {
		// Add the GNU toolchain's binary directories to Clang's binary search
		// path, so that Clang can find the correct linker. If the GNU toolchain
		// is a "system" toolchain (e.g. in /usr/bin), its binaries will already
		// be in Clang's search path, so these arguments have no effect.
		binDirs = append(binDirs, tc.gnu.getBinDirs()...)
	}

	tc.ldflags = append(tc.ldflags, utils.PrefixAll(binDirs, "-B")...)

	if useGnuLibgcc {
		dirs := utils.AppendUnique(getFileNameDir(tc.gnu, "libgcc.a"),
			getFileNameDir(tc.gnu, "libgcc_s.so"))
		tc.ldflags = append(tc.ldflags, utils.PrefixAll(dirs, "-L")...)
	}

	if useGnuStl {
		tc.cxxflags = append(tc.cxxflags,
			utils.PrefixAll(tc.gnu.getStdCxxHeaderDirs(), "-isystem ")...)
	}

	if rt == "libgcc" {
		// GCC __atomic__ builtins are provided by GNU libatomic.
		// Clang supports them via compiler-rt. However clang does not
		// link against libatomic automatically when libgcc is the
		// compiler runtime. libatomic is only needed for certain
		// architectures, so check whether it is present before trying
		// to link against it.
		//
		// libatomic is expected to be in the same dir as libgcc, so
		// the check of whether it is present must happen after adding
		// the -L for libgcc (if needed). We expect an error.
		_, err := getFileName(tc, "libatomic.so")
		if err != nil {
			tc.ldlibs = append(tc.ldlibs, "-latomic")
		}
	}

	// Combine cflags and cxxflags once here, to avoid appending during
	// every call to getCXXCompiler().
	tc.cxxflags = append(tc.cxxflags, tc.cflags...)

	tc.linker = newDefaultLinker(tc.clangxxBinary, tc.cflags, []string{})
	tc.flagCache = newFlagCache()

	return
}

func newToolchainClangNative(config *BobConfig) (tc toolchainClangNative) {
	tc.toolchainClangCommon = newToolchainClangCommon(config, TgtTypeHost)
	return
}

func newToolchainClangCross(config *BobConfig) (tc toolchainClangCross) {
	tc.toolchainClangCommon = newToolchainClangCommon(config, TgtTypeTarget)
	return
}
