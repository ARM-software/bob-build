// +build soong

/*
 * Copyright 2020 Arm Limited.
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
package kernelmodule

import (
	"fmt"
	"strings"

	"android/soong/android"
	"github.com/google/blueprint"
)

func init() {
	android.RegisterModuleType("kernel_module_bob", KernelModuleFactory)
}

var (
	pctx = android.NewPackageContext("plugins/kernelmodule")
)

type KbuildArgs struct {
	KmodBuild          string
	ExtraIncludes      string
	ExtraCflags        string
	KbuildExtraSymbols string
	KernelDir          string
	KernelCrossCompile string
	KbuildOptions      string
	MakeArgs           string
	OutputModuleDir    string
	CCFlag             string
	HostCCFlag         string
	ClangTripleFlag    string
}

func (a KbuildArgs) toDict() map[string]string {
	return map[string]string{
		"kmod_build":           a.KmodBuild,
		"extra_includes":       a.ExtraIncludes,
		"extra_cflags":         a.ExtraCflags,
		"kbuild_extra_symbols": a.KbuildExtraSymbols,
		"kernel_dir":           a.KernelDir,
		"kernel_cross_compile": a.KernelCrossCompile,
		"kbuild_options":       a.KbuildOptions,
		"make_args":            a.MakeArgs,
		"output_module_dir":    a.OutputModuleDir,
		"cc_flag":              a.CCFlag,
		"hostcc_flag":          a.HostCCFlag,
		"clang_triple_flag":    a.ClangTripleFlag,
	}
}

// dependencyTag contains the name of the tag used to track a particular type
// of dependency between modules
type kernelModuleDependencyTag struct {
	blueprint.BaseDependencyTag
}

var kernelModuleDepTag kernelModuleDependencyTag

type KernelModuleProps struct {
	// set the name of the output
	Stem string
	// list of source files to be compiled into kernel module,
	// relative to local dir
	Srcs []string
	// will resulting ninja rule be run by default
	Default bool
	// list of module names with symbols we depend on
	Extra_Symbols []string
	// if install path is not empty, module will be installed onto partition,
	// it should contain path relative to partition root
	Install_Path string
	// ninja input parameters for kmod_build.py rule
	Args KbuildArgs
}

type KernelModule struct {
	android.ModuleBase
	Properties      KernelModuleProps
	Symvers         android.WritablePath
	BuiltModule     android.WritablePath
	InstalledModule android.InstallPath
}

// implemented interfaces check
var _ android.Module = (*KernelModule)(nil)
var _ android.AndroidMkEntriesProvider = (*KernelModule)(nil)

func KernelModuleFactory() android.Module {
	m := &KernelModule{}
	// register all structs that contain module properties (parsable from .bp file)
	// note: we register our custom properties first, to take precedence before common ones
	m.AddProperties(&m.Properties)

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibCommon)

	return m
}

// set up dependant kernel modules
func (m *KernelModule) DepsMutator(mctx android.BottomUpMutatorContext) {
	mctx.AddDependency(mctx.Module(), kernelModuleDepTag, m.Properties.Extra_Symbols...)
}

const prebuiltMake = "prebuilts/build-tools/linux-x86/bin/make"

var soongKbuildRule = pctx.StaticRule(
	"bobKbuild",
	blueprint.RuleParams{
		Command: "python $kmod_build -o $out --depfile $depfile " +
			"--common-root $common_root " +
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
	},
	"kmod_build", "depfile", "extra_includes", "extra_cflags", "kbuild_extra_symbols", "kernel_dir", "kernel_cross_compile",
	"kbuild_options", "make_args", "output_module_dir", "cc_flag", "hostcc_flag", "clang_triple_flag", "common_root")

func (m *KernelModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// preserve symvers location for this module (for the parent pass)
	m.Symvers = android.PathForModuleOut(ctx, "Module.symvers")
	m.BuiltModule = android.PathForModuleOut(ctx, m.Properties.Stem+".ko")
	args := m.Properties.Args.toDict()

	// gather symvers location for all dependent kernel modules
	depSymvers := []android.Path{}
	ctx.VisitDirectDepsIf(
		func(m android.Module) bool { return ctx.OtherModuleDependencyTag(m) == kernelModuleDepTag },
		func(m android.Module) {
			if km, ok := m.(*KernelModule); ok {
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
		args["kbuild_extra_symbols"] = "--extra-symbols " + strings.Join(temp, " ")
	}

	args["output_module_dir"] = android.PathForModuleOut(ctx).String()
	args["common_root"] = ctx.ModuleDir()

	ctx.Build(pctx,
		android.BuildParams{
			Rule:        soongKbuildRule,
			Description: "kbuild " + ctx.ModuleName(),
			Inputs:      android.PathsForModuleSrc(ctx, m.Properties.Srcs),
			Implicits:   append(depSymvers, android.PathForSource(ctx, args["kmod_build"])),
			Outputs:     []android.WritablePath{m.BuiltModule},
			Args:        args,
			Default:     m.Properties.Default,
		})

	if m.Properties.Install_Path != "" {
		// generate ninja rule for copying file onto partition, also preserve install location
		m.InstalledModule = ctx.InstallFile(android.PathForModuleInstall(ctx, m.Properties.Install_Path), m.Properties.Stem+".ko", m.BuiltModule)
	}

	// Add a dependency between Module.symvers and the kernel module. This
	// should really be added to Outputs or ImplicitOutputs above, but
	// Ninja doesn't support dependency files with multiple outputs yet.
	ctx.Build(pctx,
		android.BuildParams{
			Rule:    blueprint.Phony,
			Inputs:  []android.Path{m.BuiltModule},
			Outputs: []android.WritablePath{m.Symvers},
		})
}

func (m *KernelModule) AndroidMkEntries() []android.AndroidMkEntries {
	outputFile := android.OptionalPathForPath(m.BuiltModule)
	if m.Properties.Install_Path != "" {
		// reference InstalledModule instead of BuiltModule will ensure triggering install rule after build rule
		outputFile = android.OptionalPathForPath(m.InstalledModule)
	}

	return []android.AndroidMkEntries{android.AndroidMkEntries{
		Class:      "DATA",
		OutputFile: outputFile,
		Include:    "$(BUILD_PREBUILT)",
		// don't install in data partition (which is enforced behavior when class is DATA)
		ExtraEntries: []android.AndroidMkExtraEntriesFunc{
			func(entries *android.AndroidMkEntries) {
				entries.SetBool("LOCAL_UNINSTALLABLE_MODULE", true)
			},
		},
	}}
}

// required to generate ninja rule for copying file onto partition
func (m *KernelModule) InstallBypassMake() bool {
	return true
}
