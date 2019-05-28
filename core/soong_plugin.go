// +build soong

/*
 * Copyright 2019 Arm Limited.
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

/*
 * This file is included when Bob is being run as a Soong plugin.
 *
 * The build tag on the first line ensures that it is not included in the build
 * by accident, and that it is not included in `go test` or similar checks,
 * which would fail, because Soong is not available in that environment.
 */

package core

import (
	"sync"

	"android/soong/android"
	"github.com/google/blueprint"
)

var (
	loadConfigOnce   sync.Once
	onceLoadedConfig *bobConfig
)

// During the build, Soong will do a "test" of each plugin, which loads the
// module, including calling its `init()` functions. That means that we can't
// load the config file in `init()`, because the tests would fail if it doesn't
// exist. Work around this by deferring loading the config file until a module
// factory is actually called.
func soongGetConfig() *bobConfig {
	loadConfigOnce.Do(func() {
		// TODO: This should not be hard-coded. We should probably get
		// the config path from the build dir, possibly using different
		// files depending on the product.
		jsonPath := "external/bob-build/tests/config.json"

		onceLoadedConfig = &bobConfig{}
		onceLoadedConfig.Properties = loadConfig(jsonPath)
		// TODO: This should be chosen based on the config, but hard-code it for now.
		onceLoadedConfig.Generator = &soongGenerator{}
	})
	return onceLoadedConfig

}

func getConfig(interface{}) *bobConfig {
	return soongGetConfig()
}

type moduleBase struct {
	android.ModuleBase
}

func (m *moduleBase) GenerateAndroidBuildActions(ctx android.ModuleContext) {}

type soongGenerator struct {
	toolchainSet
}

func (g *soongGenerator) aliasActions(m *alias, ctx blueprint.ModuleContext)                        {}
func (g *soongGenerator) binaryActions(*binary, blueprint.ModuleContext)                            {}
func (g *soongGenerator) genBinaryActions(*generateBinary, blueprint.ModuleContext, []inout)        {}
func (g *soongGenerator) genSharedActions(*generateSharedLibrary, blueprint.ModuleContext, []inout) {}
func (g *soongGenerator) genStaticActions(*generateStaticLibrary, blueprint.ModuleContext, []inout) {}
func (g *soongGenerator) generateSourceActions(*generateSource, blueprint.ModuleContext, []inout)   {}
func (g *soongGenerator) kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext)          {}
func (g *soongGenerator) resourceActions(*resource, blueprint.ModuleContext)                        {}
func (g *soongGenerator) sharedActions(*sharedLibrary, blueprint.ModuleContext)                     {}
func (g *soongGenerator) staticActions(*staticLibrary, blueprint.ModuleContext)                     {}
func (g *soongGenerator) transformSourceActions(*transformSource, blueprint.ModuleContext, []inout) {}

func (g *soongGenerator) buildDir() string                           { return "" }
func (g *soongGenerator) sourcePrefix() string                       { return "" }
func (g *soongGenerator) sharedLibsDir(tgt tgtType) string           { return "" }
func (g *soongGenerator) sourceOutputDir(m *generateCommon) string   { return "" }
func (g *soongGenerator) binaryOutputDir(m *binary) string           { return "" }
func (g *soongGenerator) staticLibOutputDir(m *staticLibrary) string { return "" }
func (g *soongGenerator) sharedLibOutputDir(m *sharedLibrary) string { return "" }
func (g *soongGenerator) kernelModOutputDir(m *kernelModule) string  { return "" }

func (g *soongGenerator) init(*blueprint.Context, *bobConfig) {}

func soongRegisterModule(name string, mf factoryWithConfig) {
	// Create a closure adapting Bob's module factories to the format Soong uses.
	factory := func() android.Module {
		bpModule, properties := mf(soongGetConfig())
		// This type assertion should always pass as long as every Bob
		// module type embeds moduleBase
		soongModule := bpModule.(android.Module)

		for _, property := range properties {
			soongModule.AddProperties(property)
		}

		return soongModule
	}
	android.RegisterModuleType(name, factory)
}

func init() {
	registerModuleTypes(soongRegisterModule)
}
