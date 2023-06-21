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
)

type AndroidPlatform struct {
	toolchains toolchain.ToolchainSet
	logger     *warnings.WarningLogger
	env        *config.EnvironmentVariables
}

var _ Platform = (*AndroidPlatform)(nil)

func (g *AndroidPlatform) BuildDir() string {
	// The androidbp backend writes an Android.bp file, which should
	// never reference an actual output directory (which will be
	// chosen by Soong). Therefore this function returns an empty
	// string.
	return ""
}

func (g *AndroidPlatform) SourceDir() string {
	// On the androidbp backend, sourceDir() is only used for match_src
	// handling and for locating bob scripts. In these cases we want
	// paths relative to ANDROID_BUILD_TOP directory,
	// which is where all commands will be executed from
	return g.env.SrcDir
}

func (g *AndroidPlatform) BobScriptsDir() string {
	// In the androidbp backend, we just want the relative path to the
	// script directory.
	srcToScripts, _ := filepath.Rel(g.SourceDir(), filepath.Join(g.env.BobDir, "scripts"))
	return srcToScripts
}

func (g *AndroidPlatform) SourceOutputDir(m blueprint.Module) string {
	return ""
}

func (g *AndroidPlatform) SharedLibsDir(toolchain.TgtType) string {
	// When writing link commands, it's common to put all the shared
	// libraries in a single location to make it easy for the linker to
	// find them. This function tells us where this is for the current
	// generatorBackend.
	//
	// In the androidbp backend, we don't write link command lines. Soong
	// will do this after processing the generated Android.bp. Therefore
	// this function just returns an empty string.
	return ""
}

func (g *AndroidPlatform) StaticLibOutputDir(tgt toolchain.TgtType) string {
	return ""
}

func (g *AndroidPlatform) BinaryOutputDir(toolchain.TgtType) string {
	return ""
}

func (g *AndroidPlatform) EscapeFlag(s string) string {
	// Soong will handle the escaping of flags, so the androidbp backend
	// just passes them through.
	return s
}

func (g *AndroidPlatform) GetLogger() *warnings.WarningLogger {
	return g.logger
}

func (g *AndroidPlatform) Init(config *config.Properties) {
	g.toolchains.Configure(config)
}

func (g *AndroidPlatform) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	return g.toolchains.GetToolchain(tgt)
}

func NewAndroidPlatform(env *config.EnvironmentVariables, cfg *config.Properties, logger *warnings.WarningLogger) Platform {
	p := AndroidPlatform{
		logger: logger,
		env:    env,
	}

	p.Init(cfg)

	return &p
}
