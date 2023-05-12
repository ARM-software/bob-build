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

package core

import (
	"fmt"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
)

type linker interface {
	getTool() string
	getFlags() []string
	getLibs() []string
	keepUnusedDependencies() string
	dropUnusedDependencies() string
	setRpathLink(string) string
	setVersionScript(string) string
	setRpath([]string) string
	linkWholeArchives([]string) string
	keepSharedLibraryTransitivity() string
	dropSharedLibraryTransitivity() string
	getForwardingLibFlags() string
}

type defaultLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l defaultLinker) getTool() string {
	return l.tool
}

func (l defaultLinker) getFlags() []string {
	return l.flags
}

func (l defaultLinker) getLibs() []string {
	return l.libs
}

func (l defaultLinker) keepUnusedDependencies() string {
	return "-Wl,--no-as-needed"
}

func (l defaultLinker) dropUnusedDependencies() string {
	return "-Wl,--as-needed"
}

func (l defaultLinker) keepSharedLibraryTransitivity() string {
	return "-Wl,--copy-dt-needed-entries"
}

func (l defaultLinker) dropSharedLibraryTransitivity() string {
	return "-Wl,--no-copy-dt-needed-entries"
}

func (l defaultLinker) getForwardingLibFlags() string {
	return "-fuse-ld=bfd"
}

func (l defaultLinker) setRpathLink(path string) string {
	return "-Wl,-rpath-link," + path
}

func (l defaultLinker) setVersionScript(path string) string {
	return "-Wl,--version-script," + path
}

func (l defaultLinker) setRpath(paths []string) string {
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

func (l defaultLinker) linkWholeArchives(libs []string) string {
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
