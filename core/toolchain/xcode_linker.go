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
	"github.com/ARM-software/bob-build/internal/utils"
)

type xcodeLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l xcodeLinker) GetTool() string {
	return l.tool
}

func (l xcodeLinker) GetFlags() []string {
	return l.flags
}

func (l xcodeLinker) GetLibs() []string {
	return l.libs
}

func (l xcodeLinker) KeepUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) DropUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) SetRpathLink(path string) string {
	return ""
}

func (l xcodeLinker) SetVersionScript(path string) string {
	return ""
}

func (l xcodeLinker) SetRpath(path []string) string {
	return ""
}

func (l xcodeLinker) LinkWholeArchives(libs []string) string {
	return utils.Join(libs)
}

func (l xcodeLinker) KeepSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) DropSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) GetForwardingLibFlags() string {
	return ""
}

func newXcodeLinker(tool string, flags, libs []string) (linker xcodeLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}
