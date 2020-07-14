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

package core

import (
	"path/filepath"

	"github.com/google/blueprint"

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

func (g *androidBpGenerator) kernelModuleActions(l *kernelModule, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(l) {
		return
	}

	bpmod, err := AndroidBpFile().NewModule("genrule_bob", l.Name())
	if err != nil {
		panic(err)
	}

	// Calculate and record outputs
	out := l.outputName() + ".ko"
	l.outs = []string{out}

	kmod_build := getBackendPathInBobScriptsDir(g, "kmod_build.py")

	sources_param := "${in}"
	var module_deps []string
	for _, mod := range l.extraSymbolsModules(mctx) {
		module_deps = append(module_deps, mod.Name())
		// reference all dependent modules outputs, needed for related symvers files
		sources_param += " ${" + mod.Name() + "_dir}/Module.symvers"
	}

	kdir := l.Properties.Kernel_dir
	if !filepath.IsAbs(kdir) {
		kdir = getPathInSourceDir(kdir)
	}

	addProvenanceProps(bpmod, l.Properties.AndroidProps)
	bpmod.AddStringList("srcs", l.Properties.getSources(mctx))
	bpmod.AddStringList("module_deps", module_deps)
	bpmod.AddStringList("out", l.outs)
	bpmod.AddStringList("implicit_outs", []string{"Module.symvers"})
	bpmod.AddString("tool", kmod_build)
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
			"--extra-cflags='" + utils.Join(l.Properties.Cflags) + "'",
		},
		stringParam("--kbuild-options", utils.Join(l.Properties.Kbuild_options)),
		stringParam("--cross-compile", l.Properties.Kernel_cross_compile),
		stringParam("--cc", l.Properties.Kernel_cc),
		stringParam("--hostcc", l.Properties.Kernel_hostcc),
		stringParam("--clang-triple", l.Properties.Kernel_clang_triple),
		stringParam("--ld", l.Properties.Kernel_ld),
		stringParams("-I",
			l.Properties.Include_dirs,
			getPathsInSourceDir(l.Properties.Local_include_dirs)),
		l.Properties.Make_args,
	)

	installBase, installRel, ok := getSoongInstallPath(l.getInstallableProps())
	if ok {
		switch installBase {
		case "data":
			bpmod.AddBool("install_in_data", true)
		case "tests":
			/* Eventually we want to install in testcases,
			 * but we can't put binaries there yet:
			 * bpmod.AddBool("install_in_testcases", true)
			 * So place resources in /data/nativetest to align with cc_test.
			 *
			 * `nativetest` has no corresponding `InstallIn...` method,
			 * so request the `/data` partition and add the `nativetest`
			 * part in as another relative component. */
			bpmod.AddBool("install_in_data", true)
			installRel = filepath.Join(installRel, "nativetest")
		default:
			/* Paths like `lib/modules` are implicitly in /system, or /vendor, but
			 * unlike e.g. a library, which would add the `lib` for us, we need to add
			 * it ourselves here - so the whole path is used as the relative part. */
			installRel = filepath.Join(installBase, installRel)
		}
		bpmod.AddString("install_path", installRel)
	}

}
