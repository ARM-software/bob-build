/*
 * Copyright 2018-2019 Arm Limited.
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
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

type kernelModule struct {
	blueprint.SimpleName
	Properties struct {
		Features
		Build
		Defaults []string
	}
}

func (m *kernelModule) defaults() []string {
	return m.Properties.Defaults
}

func (m *kernelModule) build() *Build {
	return &m.Properties.Build
}

func (m *kernelModule) topLevelProperties() []interface{} {
	return []interface{}{&m.Properties.Build.BuildProps}
}

func (m *kernelModule) features() *Features {
	return &m.Properties.Features
}

func (m *kernelModule) outputName() string {
	if len(m.Properties.Out) > 0 {
		return m.Properties.Out
	}
	return m.SimpleName.Name()
}

func (m *kernelModule) altName() string {
	return m.outputName()
}

func (m *kernelModule) altShortName() string {
	return m.altName()
}

func (m *kernelModule) shortName() string {
	return m.Name()
}

func (m *kernelModule) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *kernelModule) outputDir(g generatorBackend) string {
	return g.kernelModOutputDir(m)
}

func (m *kernelModule) outputs(g generatorBackend) []string {
	return []string{filepath.Join(m.outputDir(g), m.outputName()+".ko")}
}

func (m *kernelModule) implicitOutputs(g generatorBackend) []string {
	return []string{}
}

func (m *kernelModule) filesToInstall(ctx blueprint.ModuleContext) []string {
	return m.outputs(getBackend(ctx))
}

func (m *kernelModule) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *kernelModule) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, installDepTag, kernelModuleDepTag)
}

func (m *kernelModule) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.Build.processPaths(ctx)
}

func (m *kernelModule) extraSymbolsFiles(ctx blueprint.ModuleContext) (files []string) {
	g := getBackend(ctx)

	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == kernelModuleDepTag },
		func(m blueprint.Module) {
			if km, ok := m.(*kernelModule); ok {
				file := filepath.Join(km.outputDir(g), "Module.symvers")
				files = append(files, file)
			} else {
				panic(fmt.Errorf("invalid extra_symbols, %s not a kernel module", ctx.OtherModuleName(m)))
			}
		})

	return
}

func (m *kernelModule) generateKbuildArgs(ctx blueprint.ModuleContext) map[string]string {
	var extraIncludePaths []string

	g := getBackend(ctx)

	extraCflags := m.build().BuildProps.Cflags

	for _, includeDir := range m.build().BuildProps.Local_include_dirs {
		includeDir = "-I" + filepath.Join(g.sourcePrefix(), includeDir)
		extraIncludePaths = append(extraIncludePaths, includeDir)
	}

	for _, includeDir := range m.build().BuildProps.Include_dirs {
		includeDir = "-I" + includeDir
		extraIncludePaths = append(extraIncludePaths, includeDir)
	}

	kmodBuild := filepath.Join(bobdir, "scripts", "kmod_build.py")
	kdir := m.Properties.Build.Kernel_dir
	if !filepath.IsAbs(kdir) {
		kdir = filepath.Join(g.sourcePrefix(), kdir)
	}

	extraSymbols := ""
	if len(m.extraSymbolsFiles(ctx)) > 0 {
		extraSymbols = "--extra-symbols " + strings.Join(m.extraSymbolsFiles(ctx), " ")
	}

	kbuildOptions := ""
	if len(m.build().Kbuild_options) > 0 {
		kbuildOptions = "--kbuild-options " + strings.Join(m.build().Kbuild_options, " ")
	}

	hostToolchain := m.Properties.Build.Kernel_hostcc
	if hostToolchain != "" {
		hostToolchain = "--hostcc " + hostToolchain
	}

	kernelToolchain := m.Properties.Build.Kernel_cc
	if kernelToolchain != "" {
		kernelToolchain = "--cc " + kernelToolchain
	}

	clangTriple := m.Properties.Build.Kernel_clang_triple
	if clangTriple != "" {
		clangTriple = "--clang-triple " + clangTriple
	}

	return map[string]string{
		"kmod_build":           kmodBuild,
		"extra_includes":       strings.Join(extraIncludePaths, " "),
		"extra_cflags":         strings.Join(extraCflags, " "),
		"kbuild_extra_symbols": extraSymbols,
		"kernel_dir":           kdir,
		"kernel_cross_compile": m.Properties.Build.Kernel_cross_compile,
		"kbuild_options":       kbuildOptions,
		"make_args":            strings.Join(m.Properties.Build.Make_args, " "),
		"output_module_dir":    filepath.Join(m.outputDir(g), ctx.ModuleDir()),
		"cc_flag":              kernelToolchain,
		"hostcc_flag":          hostToolchain,
		"clang_triple_flag":    clangTriple,
	}
}

func (m *kernelModule) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		ctx.Config().(*bobConfig).Generator.kernelModuleActions(m, ctx)
	}
}

func kernelModuleFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &kernelModule{}
	availableFeatures := config.getAvailableFeatures()
	module.Properties.Features.Init(availableFeatures, BuildProps{})
	module.Properties.Build.Target.Init(availableFeatures, BuildProps{})
	module.Properties.Build.Host.Init(availableFeatures, BuildProps{})

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
