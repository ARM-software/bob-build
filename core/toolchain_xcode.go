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

type toolchainXcode struct {
	arBinary    string
	asBinary    string
	dsymBinary  string
	stripBinary string
	otoolBinary string
	nmBinary    string
	ccBinary    string
	cxxBinary   string
	linker      linker
	prefix      string
	target      string
	flagCache   *flagSupportedCache

	cflags  []string
	ldflags []string
}

type toolchainXcodeNative struct {
	toolchainXcode
}

type toolchainXcodeCross struct {
	toolchainXcode
}

func (tc toolchainXcode) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainXcode) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainXcode) getCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainXcode) getCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

func (tc toolchainXcode) getLinker() linker {
	return tc.linker
}

func (tc toolchainXcode) getStripFlags() []string {
	return []string{
		"--format", "macho",
		"--dsymutil-tool", tc.dsymBinary,
		"--strip-tool", tc.stripBinary,
	}
}

func (tc toolchainXcode) getLibraryTocFlags() []string {
	return []string{
		"--format", "macho",
		"--otool-tool", tc.otoolBinary,
		"--nm-tool", tc.nmBinary,
	}
}

func (tc toolchainXcode) checkFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func newToolchainXcodeCommon(config *BobConfig, tgt TgtType) (tc toolchainXcode) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_xcode_prefix")
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")
	tc.dsymBinary = props.GetString(string(tgt) + "_dsymutil_binary")
	tc.stripBinary = props.GetString(string(tgt) + "_strip_binary")
	tc.otoolBinary = props.GetString(string(tgt) + "_otool_binary")
	tc.nmBinary = props.GetString(string(tgt) + "_nm_binary")

	tc.ccBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cc_binary")
	tc.cxxBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cxx_binary")

	tc.target = props.GetString(string(tgt) + "_xcode_triple")

	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
		tc.ldflags = append(tc.ldflags, "-target", tc.target)
	}

	tc.linker = newXcodeLinker(tc.cxxBinary, tc.ldflags, []string{})
	tc.flagCache = newFlagCache()

	return
}

func newToolchainXcodeNative(config *BobConfig) (tc toolchainXcodeNative) {
	tc.toolchainXcode = newToolchainXcodeCommon(config, TgtTypeHost)
	return
}

func newToolchainXcodeCross(config *BobConfig) (tc toolchainXcodeCross) {
	tc.toolchainXcode = newToolchainXcodeCommon(config, TgtTypeTarget)
	return
}
