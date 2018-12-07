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
	"strings"
)

type toolchain interface {
	getAssembler() (tool string, flags []string)
	getCCompiler() (tool string, flags []string)
	getCXXCompiler() (tool string, flags []string)
	getArchiver() (tool string, flags []string)
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

func newToolchainGnuNative(config *bobConfig) (tc toolchainGnuNative) {
	props := config.Properties
	tc.arBinary = props.GetString("ar_binary")
	tc.asBinary = props.GetString("as_binary")
	tc.gccBinary = props.GetString("gcc_binary")
	tc.gxxBinary = props.GetString("gxx_binary")
	return
}

func newToolchainGnuCross(config *bobConfig) (tc toolchainGnuCross) {
	props := config.Properties
	tc.prefix = props.GetString("toolchain_prefix")
	tc.arBinary = tc.prefix + props.GetString("ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")
	tc.gccBinary = tc.prefix + props.GetString("gcc_binary")
	tc.gxxBinary = tc.prefix + props.GetString("gxx_binary")
	tc.cflags = strings.Split(props.GetString("gcc_target_flags"), " ")
	return
}

type toolchainClangCommon struct {
	// Options read from the config:
	clangBinary   string
	clangxxBinary string

	// Use the GNU toolchain's 'ar' and 'as'
	gnu toolchain

	// Calculated during toolchain initialization:
	cflags   []string // Flags for both C and C++
	cxxflags []string // Flags just for C++
}

type toolchainClangNative struct {
	toolchainClangCommon
}

type toolchainClangCross struct {
	toolchainClangCommon
	target           string
	sysroot          string
	toolchainVersion string
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

func newToolchainClangCommon(config *bobConfig, gnu toolchain) (tc toolchainClangCommon) {
	props := config.Properties
	tc.clangBinary = props.GetString("clang_binary")
	tc.clangxxBinary = props.GetString("clangxx_binary")
	tc.gnu = gnu
	return
}

func newToolchainClangNative(config *bobConfig) (tc toolchainClangNative) {
	gnu := newToolchainGnuNative(config)
	tc.toolchainClangCommon = newToolchainClangCommon(config, gnu)
	return
}

func newToolchainClangCross(config *bobConfig) (tc toolchainClangCross) {
	gnu := newToolchainGnuCross(config)
	tc.toolchainClangCommon = newToolchainClangCommon(config, gnu)

	props := config.Properties
	tc.target = props.GetString("clang_target")
	tc.sysroot = props.GetString("clang_sysroot")
	tc.toolchainVersion = props.GetString("target_toolchain_version")

	if tc.sysroot != "" {
		if tc.target == "" {
			panic(errors.New("CLANG_TARGET is not set"))
		}
		if tc.toolchainVersion == "" {
			panic(errors.New("TARGET_TOOLCHAIN_VERSION is not set"))
		}
		tc.cflags = append(tc.cflags, "--sysroot", tc.sysroot)

		tc.cxxflags = append(tc.cxxflags,
			"-isystem", fmt.Sprintf("%s/../include/c++/%s",
				tc.sysroot, tc.toolchainVersion),
			"-isystem", fmt.Sprintf("%s/../include/c++/%s/%s",
				tc.sysroot, tc.toolchainVersion,
				tc.target))
	}
	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
	}

	// Combine cflags and cxxflags once here, to avoid appending during
	// every call to getCXXCompiler().
	tc.cxxflags = append(tc.cflags, tc.cxxflags...)

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

	if props.GetBool("toolchain_clang") {
		tcs.host = newToolchainClangNative(config)
		tcs.target = newToolchainClangCross(config)
	} else if props.GetBool("toolchain_gnu") {
		tcs.host = newToolchainGnuNative(config)
		tcs.target = newToolchainGnuCross(config)
	} else {
		panic(errors.New("no usable compiler toolchain configured"))
	}
}
