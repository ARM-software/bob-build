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
	"strings"
)

type toolchainArmClang struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	objdumpBinary string
	ccBinary      string
	cxxBinary     string
	linker        linker
	prefix        string
	cflags        []string // Flags for both C and C++
	flagCache     *flagSupportedCache

	is64BitOnly bool
}

type toolchainArmClangNative struct {
	toolchainArmClang
}

type toolchainArmClangCross struct {
	toolchainArmClang
}

func (tc toolchainArmClang) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainArmClang) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainArmClang) getCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainArmClang) getCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

func (tc toolchainArmClang) getLinker() linker {
	return tc.linker
}

func (tc toolchainArmClang) getStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainArmClang) getLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainArmClang) checkFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func (tc toolchainArmClang) Is64BitOnly() bool {
	return tc.is64BitOnly
}

func newToolchainArmClangCommon(config *BobConfig, tgt TgtType) (tc toolchainArmClang) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_gnu_prefix")
	tc.arBinary = tc.prefix + props.GetString("armclang_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("armclang_as_binary")
	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")
	tc.ccBinary = tc.prefix + props.GetString(string(tgt)+"_armclang_cc_binary")
	tc.cxxBinary = tc.prefix + props.GetString(string(tgt)+"_armclang_cxx_binary")
	tc.linker = newDefaultLinker(tc.cxxBinary, []string{}, []string{})

	tc.cflags = strings.Split(config.Properties.GetString(string(tgt)+"_armclang_flags"), " ")
	tc.flagCache = newFlagCache()
	tc.is64BitOnly = props.GetBool(string(tgt) + "_64bit_only")

	return
}

func newToolchainArmClangNative(config *BobConfig) (tc toolchainArmClangNative) {
	tc.toolchainArmClang = newToolchainArmClangCommon(config, TgtTypeHost)
	return
}

func newToolchainArmClangCross(config *BobConfig) (tc toolchainArmClangCross) {
	tc.toolchainArmClang = newToolchainArmClangCommon(config, TgtTypeTarget)
	return
}
