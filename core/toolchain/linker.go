/*
 * Copyright 2023 Arm Limited.
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

package toolchain

import (
	"fmt"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
)

type Linker interface {
	GetTool() string
	GetFlags() []string
	GetLibs() []string
	KeepUnusedDependencies() string
	DropUnusedDependencies() string
	SetRpathLink(string) string
	SetVersionScript(string) string
	SetRpath([]string) string
	LinkWholeArchives([]string) string
	KeepSharedLibraryTransitivity() string
	DropSharedLibraryTransitivity() string
	GetForwardingLibFlags() string
}

type defaultLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l defaultLinker) GetTool() string {
	return l.tool
}

func (l defaultLinker) GetFlags() []string {
	return l.flags
}

func (l defaultLinker) GetLibs() []string {
	return l.libs
}

func (l defaultLinker) KeepUnusedDependencies() string {
	return "-Wl,--no-as-needed"
}

func (l defaultLinker) DropUnusedDependencies() string {
	return "-Wl,--as-needed"
}

func (l defaultLinker) KeepSharedLibraryTransitivity() string {
	return "-Wl,--copy-dt-needed-entries"
}

func (l defaultLinker) DropSharedLibraryTransitivity() string {
	return "-Wl,--no-copy-dt-needed-entries"
}

func (l defaultLinker) GetForwardingLibFlags() string {
	return "-fuse-ld=bfd"
}

func (l defaultLinker) SetRpathLink(path string) string {
	return "-Wl,-rpath-link," + path
}

func (l defaultLinker) SetVersionScript(path string) string {
	return "-Wl,--version-script," + path
}

func (l defaultLinker) SetRpath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("-Wl,--enable-new-dtags")
	for _, p := range paths {
		fmt.Fprintf(&b, ",-rpath=%s", p)
	}
	return b.String()
}

func (l defaultLinker) LinkWholeArchives(libs []string) string {
	if len(libs) == 0 {
		return ""
	}
	return fmt.Sprintf("-Wl,--whole-archive %s -Wl,--no-whole-archive", utils.Join(libs))
}

func newDefaultLinker(tool string, flags, libs []string) (linker defaultLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}
