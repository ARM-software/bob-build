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
	bpmod, err := AndroidBpFile().NewModule("genrule", l.Name())
	if err != nil {
		panic(err)
	}

	out := l.outputName() + ".ko"

	kmod_build, _ := filepath.Rel(srcdir, filepath.Join(bobdir, "scripts", "kmod_build.py"))

	srcs := l.Properties.getSources(mctx)
	for _, mod := range l.extraSymbolsModules(mctx) {
		srcs = append(srcs, ":"+mod.Name())
	}

	kdir := l.Properties.Kernel_dir
	if !filepath.IsAbs(kdir) {
		kdir = filepath.Join(srcdir, kdir)
	}

	addProvenanceProps(bpmod, l.Properties.AndroidProps)
	bpmod.AddStringList("srcs", srcs)
	bpmod.AddStringList("out", []string{out, "Module.symvers"})
	bpmod.AddStringList("tool_files", []string{kmod_build})
	bpmod.AddBool("depfile", true)

	// Generate the build command. Use the `stringParam` helper for options which
	// may be empty to avoid writing a flag name with no corresponding value.
	bpmod.AddStringCmd("cmd",
		[]string{
			"python", "$(location " + kmod_build + ")",
			"-o", filepath.Join("$(genDir)", out),
			"--depfile", "$(depfile)",
			"--sources", "$(in)",
			"--common-root", srcdir,
			"--kernel", kdir,
			"--module-dir", "$(genDir)/" + mctx.ModuleDir(),
			"--make-command", prebuiltMake,
		},
		stringParam("--cc", l.Properties.Kernel_cc),
		stringParam("--clang-triple", l.Properties.Kernel_clang_triple),
		stringParam("--cross-compile", l.Properties.Kernel_cross_compile),
		stringParam("--hostcc", l.Properties.Kernel_hostcc),
		stringParams("-I",
			l.Properties.Include_dirs,
			utils.PrefixDirs(l.Properties.Local_include_dirs, srcdir)))
}
