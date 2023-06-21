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

// The backend package provides platform specific configuration for the generator.
// Based on the type of generator used, different paths will be required.
package backend

import (
	"sync"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

// Backend platform singleton.
// Abstracts platform specific (AOSP, Ninja) parameters and stores toolchains.
type Platform interface {
	BuildDir() string
	SourceDir() string
	BobScriptsDir() string
	SourceOutputDir(blueprint.Module) string
	SharedLibsDir(tgt toolchain.TgtType) string
	StaticLibOutputDir(tgt toolchain.TgtType) string
	BinaryOutputDir(tgt toolchain.TgtType) string
	EscapeFlag(string) string
	Init(*config.Properties)
	GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain
	GetLogger() *warnings.WarningLogger
}

var platform Platform
var lock = &sync.Mutex{}

func Get() Platform {
	if platform == nil {
		lock.Lock()
		defer lock.Unlock()
		if platform == nil {
			panic("The Backend Platform has not been configured, please call Setup() before use.")
		}
	}

	return platform
}

func Setup(env *config.EnvironmentVariables, cfg *config.Properties, logger *warnings.WarningLogger) {
	if platform == nil {
		lock.Lock()
		defer lock.Unlock()
		if platform == nil {
			switch {
			case cfg.GetBool("builder_ninja"):
				platform = NewLinuxPlatform(env, cfg, logger)
			case cfg.GetBool("builder_android_bp"):
				platform = NewAndroidPlatform(env, cfg, logger)
			default:
				utils.Die("Unknown builder backend")
			}
		}
	}
}
