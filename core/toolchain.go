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

type toolchainGnuNative struct {
	arBinary  string
	asBinary  string
	gccBinary string
	gxxBinary string
}

type toolchainGnuCross struct {
	toolchainGnuNative
	prefix      string
	targetFlags []string
}

func (tc toolchainGnuNative) getArchiver() (tool string, flags []string) {
	tool = tc.arBinary
	return
}

func (tc toolchainGnuCross) getArchiver() (tool string, flags []string) {
	nativeTool, nativeFlags := tc.toolchainGnuNative.getArchiver()
	tool = tc.prefix + nativeTool
	flags = nativeFlags
	return
}

func (tc toolchainGnuNative) getAssembler() (tool string, flags []string) {
	tool = tc.asBinary
	return
}

func (tc toolchainGnuCross) getAssembler() (tool string, flags []string) {
	nativeTool, nativeFlags := tc.toolchainGnuNative.getAssembler()
	tool = tc.prefix + nativeTool
	flags = nativeFlags
	return
}

func (tc toolchainGnuNative) getCCompiler() (tool string, flags []string) {
	tool = tc.gccBinary
	return
}

func (tc toolchainGnuCross) getCCompiler() (tool string, flags []string) {
	nativeTool, nativeFlags := tc.toolchainGnuNative.getCCompiler()
	tool = tc.prefix + nativeTool
	flags = append(nativeFlags, tc.targetFlags...)
	return
}

func (tc toolchainGnuNative) getCXXCompiler() (tool string, flags []string) {
	tool = tc.gxxBinary
	return
}

func (tc toolchainGnuCross) getCXXCompiler() (tool string, flags []string) {
	nativeTool, nativeFlags := tc.toolchainGnuNative.getCXXCompiler()
	tool = tc.prefix + nativeTool
	flags = append(nativeFlags, tc.targetFlags...)
	return
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
	tc.toolchainGnuNative = newToolchainGnuNative(config)
	tc.prefix = props.GetString("toolchain_prefix")
	tc.targetFlags = strings.Split(props.GetString("gcc_target_flags"), " ")
	return
}

type toolchainClangNative struct {
	clangBinary   string
	clangxxBinary string
	// Use the GNU toolchain's 'ar' and 'as'
	gnu toolchainGnuNative
}

type toolchainClangCross struct {
	toolchainClangNative
	target           string
	sysroot          string
	toolchainVersion string
	// Use the GNU toolchain's 'ar' and 'as'
	gnu toolchainGnuCross
}

func (tc toolchainClangNative) getArchiver() (string, []string) {
	return tc.gnu.getArchiver()
}

func (tc toolchainClangCross) getArchiver() (string, []string) {
	return tc.gnu.getArchiver()
}

func (tc toolchainClangNative) getAssembler() (string, []string) {
	return tc.gnu.getAssembler()
}

func (tc toolchainClangCross) getAssembler() (string, []string) {
	return tc.gnu.getAssembler()
}

func (tc toolchainClangNative) getCCompiler() (string, []string) {
	return tc.clangBinary, []string{}
}

func (tc toolchainClangCross) getCCompiler() (tool string, flags []string) {
	tool, flags = tc.toolchainClangNative.getCCompiler()

	if tc.target != "" {
		flags = append(flags, "-target", tc.target)
	}
	if tc.sysroot != "" {
		flags = append(flags, "--sysroot", tc.sysroot)
	}

	return tool, flags
}

func (tc toolchainClangNative) getCXXCompiler() (string, []string) {
	return tc.clangxxBinary, []string{}
}

func (tc toolchainClangCross) getCXXCompiler() (tool string, flags []string) {
	tool, flags = tc.toolchainClangNative.getCXXCompiler()

	if tc.target != "" {
		flags = append(flags, "-target", tc.target)
	}
	if tc.sysroot != "" {
		flags = append(flags,
			"--sysroot", tc.sysroot,
			"-isystem", fmt.Sprintf("%s/../include/c++/%s",
				tc.sysroot, tc.toolchainVersion),
			"-isystem", fmt.Sprintf("%s/../include/c++/%s/%s",
				tc.sysroot, tc.toolchainVersion,
				tc.target))
	}

	return tool, flags
}

func newToolchainClangNative(config *bobConfig) (tc toolchainClangNative) {
	props := config.Properties
	tc.clangBinary = props.GetString("clang_binary")
	tc.clangxxBinary = props.GetString("clangxx_binary")

	tc.gnu = newToolchainGnuNative(config)

	return
}

func newToolchainClangCross(config *bobConfig) (tc toolchainClangCross) {
	props := config.Properties
	tc.toolchainClangNative = newToolchainClangNative(config)
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
	}

	tc.gnu = newToolchainGnuCross(config)

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
