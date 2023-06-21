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

package backend

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type LinuxPlatform struct {
	toolchains toolchain.ToolchainSet
	logger     *warnings.WarningLogger
}

var _ Platform = (*LinuxPlatform)(nil)

func (g *LinuxPlatform) BuildDir() string {
	return "${BuildDir}"
}

func (g *LinuxPlatform) SourceDir() string {
	return "${SrcDir}"
}

func (g *LinuxPlatform) BobScriptsDir() string {
	return "${BobScriptsDir}"
}

func (g *LinuxPlatform) SourceOutputDir(m blueprint.Module) string {
	return filepath.Join("${BuildDir}", "gen", m.Name())
}

func (g *LinuxPlatform) SharedLibsDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "shared")
}

func (g *LinuxPlatform) StaticLibOutputDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "static")
}

func (g *LinuxPlatform) BinaryOutputDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "executable")
}

func (g *LinuxPlatform) EscapeFlag(s string) string {
	return proptools.NinjaAndShellEscape(s)
}

func (g *LinuxPlatform) Init(config *config.Properties) {
	g.toolchains.Configure(config)
}

func (g *LinuxPlatform) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	return g.toolchains.GetToolchain(tgt)
}

func (g *LinuxPlatform) GetLogger() *warnings.WarningLogger {
	return g.logger
}

func NewLinuxPlatform(_ *config.EnvironmentVariables, cfg *config.Properties, logger *warnings.WarningLogger) Platform {
	l := LinuxPlatform{
		logger: logger,
	}

	l.Init(cfg)

	return &l
}
