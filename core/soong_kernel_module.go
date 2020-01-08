// +build soong

/*
 * Copyright 2019-2020 Arm Limited.
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

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/abstr"
	"github.com/ARM-software/bob-build/utils"
)

type kernelModuleBackendProps struct {
	Stem          string
	Srcs          []string
	Args          kbuildArgs
	Default       bool
	Extra_Symbols []string
	Install_Path  string
}

type kernelModuleBackend struct {
	android.ModuleBase
	Properties      kernelModuleBackendProps
	Symvers         android.WritablePath
	BuiltModule     android.WritablePath
	InstalledModule android.InstallPath
}

func kernelModuleBackendFactory() android.Module {
	m := &kernelModuleBackend{}
	// register all structs that contain module properties (parsable from .bp file)
	// note: we register our custom properties first, to take precedence before common ones
	m.AddProperties(&m.Properties)

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)

	return m
}

func (m *kernelModule) soongBuildActions(mctx android.TopDownMutatorContext) {
	g := getBackend(mctx)

	nameProps := nameProps{
		proptools.StringPtr(m.buildbpName()),
	}

	provenanceProps := getProvenanceProps(&m.Properties.Build.BuildProps)

	installProps := m.getInstallableProps()
	installPath, ok := installProps.getInstallGroupPath()
	if !ok {
		installPath = ""
	} else {
		if installProps.Relative_install_path != nil {
			installPath = filepath.Join(installPath, proptools.String(installProps.Relative_install_path))
		}
	}

	props := kernelModuleBackendProps{
		Stem:          m.outputName(),
		Args:          m.generateKbuildArgs(mctx),
		Srcs:          utils.PrefixDirs(m.Properties.getSources(mctx), g.sourcePrefix()),
		Default:       isBuiltByDefault(m),
		Extra_Symbols: m.Properties.Extra_symbols,
		Install_Path:  installPath,
	}

	// create module and fill all its registered properties with data from prepared structs
	mctx.CreateModule(kernelModuleBackendFactory, &nameProps, provenanceProps, &props)
}

func (m *kernelModuleBackend) DepsMutator(mctx android.BottomUpMutatorContext) {
	mctx.AddDependency(mctx.Module(), kernelModuleDepTag, m.Properties.Extra_Symbols...)
}

const prebuiltMake = "prebuilts/build-tools/linux-x86/bin/make"

var soongKbuildRule = apctx.StaticRule("bobKbuild",
	blueprint.RuleParams{
		Command: "python $kmod_build -o $out --depfile $depfile " +
			"--common-root " + srcdir + " " +
			"--make-command " + prebuiltMake + " " +
			"--module-dir $output_module_dir $extra_includes " +
			"--sources $in $kbuild_extra_symbols " +
			"--kernel $kernel_dir --cross-compile '$kernel_cross_compile' " +
			"$cc_flag $hostcc_flag $clang_triple_flag " +
			"$kbuild_options --extra-cflags='$extra_cflags' $make_args",
		Depfile:     "$out.d",
		Deps:        blueprint.DepsGCC,
		Pool:        blueprint.Console,
		Description: "$out",
	}, "kmod_build", "depfile", "extra_includes", "extra_cflags", "kbuild_extra_symbols", "kernel_dir", "kernel_cross_compile",
	"kbuild_options", "make_args", "output_module_dir", "cc_flag", "hostcc_flag", "clang_triple_flag")

func (m *kernelModuleBackend) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// preserve symvers location for this module (for the parent pass)
	m.Symvers = android.PathForModuleOut(ctx, "Module.symvers")
	m.BuiltModule = android.PathForModuleOut(ctx, m.Properties.Stem+".ko")

	// gather symvers location for all dependant kernel modules
	depSymvers := []android.Path{}
	abstr.VisitDirectDepsIf(ctx,
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == kernelModuleDepTag },
		func(m blueprint.Module) {
			if km, ok := m.(*kernelModuleBackend); ok {
				depSymvers = append(depSymvers, km.Symvers)
			} else {
				panic(fmt.Errorf("%s not a kernel module backend", m.Name()))
			}
		})

	if len(depSymvers) > 0 {
		// convert to strings
		temp := []string{}
		for _, path := range depSymvers {
			temp = append(temp, path.String())
		}
		// overwrite incorrect paths
		m.Properties.Args.KbuildExtraSymbols = "--extra-symbols " + strings.Join(temp, " ")
	}

	// The kernel module builder replicates the out-of-tree module's source tree structure.
	// The kernel module will be at its equivalent position in the output tree.
	m.Properties.Args.OutputModuleDir = android.PathForModuleOut(ctx, projectModuleDir(ctx)).String()

	ctx.Build(apctx,
		android.BuildParams{
			Rule:        soongKbuildRule,
			Description: "kbuild " + ctx.ModuleName(),
			Inputs:      android.PathsForSource(ctx, m.Properties.Srcs),
			Implicits:   append(depSymvers, android.PathForSource(ctx, m.Properties.Args.KmodBuild)),
			Outputs:     []android.WritablePath{m.BuiltModule},
			Args:        m.Properties.Args.toDict(),
			Default:     m.Properties.Default,
		})

	if m.Properties.Install_Path != "" {
		// generate ninja rule for copying file onto partition, also preserve install location
		m.InstalledModule = ctx.InstallFile(android.PathForModuleInstall(ctx, m.Properties.Install_Path), m.Properties.Stem+".ko", m.BuiltModule)
	}

	// Add a dependency between Module.symvers and the kernel module. This
	// should really be added to Outputs or ImplicitOutputs above, but
	// Ninja doesn't support dependency files with multiple outputs yet.
	ctx.Build(apctx,
		android.BuildParams{
			Rule:    blueprint.Phony,
			Inputs:  []android.Path{m.BuiltModule},
			Outputs: []android.WritablePath{m.Symvers},
		})
}

func (m *kernelModuleBackend) AndroidMkEntries() android.AndroidMkEntries {
	outputFile := android.OptionalPathForPath(m.BuiltModule)
	if m.Properties.Install_Path != "" {
		// reference InstalledModule instead of BuiltModule will ensure triggering install rule after build rule
		outputFile = android.OptionalPathForPath(m.InstalledModule)
	}

	return android.AndroidMkEntries{
		Class:      "DATA",
		OutputFile: outputFile,
		Include:    "$(BUILD_PREBUILT)",
		// don't install in data partition (which is enforced behavior when class is DATA)
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(entries *android.AndroidMkEntries) {
				entries.SetBool("LOCAL_UNINSTALLABLE_MODULE", true)
			},
		},
	}
}

// required to generate ninja rule for copying file onto partition
func (m *kernelModuleBackend) InstallBypassMake() bool {
	return true
}
