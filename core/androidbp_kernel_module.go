/*
 * Copyright 2020-2023 Arm Limited.
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
	"path/filepath"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/utils"
)

func stringParam(optName string, optValue string) (opts []string) {
	if optValue != "" {
		opts = []string{optName, optValue}
	}
	return
}

func stringParams(optName string, optValueLists ...[]string) (opts []string) {
	for _, optValueList := range optValueLists {
		for _, optValue := range optValueList {
			opts = append(opts, optName)
			opts = append(opts, optValue)
		}
	}
	return
}

const prebuiltMake = "prebuilts/build-tools/linux-x86/bin/make"

func (g *androidBpGenerator) kernelModuleActions(ko *ModuleKernelObject, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(ko) {
		return
	}

	bpmod, err := AndroidBpFile().NewModule("genrule_bob", ko.Name())
	if err != nil {
		panic(err)
	}

	// Calculate and record outputs
	out := ko.outputName() + ".ko"
	ko.outs = []string{out}

	kmod_build := getBackendPathInBobScriptsDir(g, "kmod_build.py")

	sources_param := "${in}"
	var generated_deps []string
	for _, mod := range ko.extraSymbolsModules(mctx) {
		generated_deps = append(generated_deps, mod.Name())
		// reference all dependent modules outputs, needed for related symvers files
		sources_param += " $$(dirname ${" + mod.Name() + "_out})/Module.symvers"
	}

	kdir := proptools.String(ko.Properties.Kernel_dir)
	if !filepath.IsAbs(kdir) {
		kdir = getPathInSourceDir(kdir)
	}

	addProvenanceProps(bpmod, ko.Properties.AndroidProps)

	srcs := []string{}
	ko.Properties.GetSrcs(mctx).ForEach(
		func(fp filePath) bool {
			srcs = append(srcs, fp.localPath())
			return true
		})

	bpmod.AddStringList("srcs", srcs)
	bpmod.AddStringList("generated_deps", generated_deps)
	bpmod.AddStringList("out", ko.outs)
	bpmod.AddStringList("implicit_outs", []string{"Module.symvers"})
	bpmod.AddStringList("tools", []string{kmod_build})
	bpmod.AddBool("depfile", true)

	// Generate the build command. Use the `stringParam` helper for options which
	// may be empty to avoid writing a flag name with no corresponding value.
	bpmod.AddStringCmd("cmd",
		[]string{
			"${tool}",
			"-o ${out}",
			"--depfile", "${depfile}",
			"--sources", sources_param,
			"--common-root", getSourceDir(),
			"--kernel", kdir,
			"--module-dir", "${gen_dir}/" + mctx.ModuleDir(),
			"--make-command", prebuiltMake,
			"--extra-cflags='" + utils.Join(ko.Properties.Cflags) + "'",
		},
		stringParam("--kbuild-options", utils.Join(ko.Properties.Kbuild_options)),
		stringParam("--cross-compile", proptools.String(ko.Properties.Kernel_cross_compile)),
		stringParam("--cc", proptools.String(ko.Properties.Kernel_cc)),
		stringParam("--hostcc", proptools.String(ko.Properties.Kernel_hostcc)),
		stringParam("--clang-triple", proptools.String(ko.Properties.Kernel_clang_triple)),
		stringParam("--ld", proptools.String(ko.Properties.Kernel_ld)),
		stringParams("-I",
			ko.Properties.Include_dirs,
			getPathsInSourceDir(ko.Properties.Local_include_dirs)),
		ko.Properties.Make_args,
	)

	addInstallProps(bpmod, ko.getInstallableProps(), ko.Properties.isProprietary())
}
