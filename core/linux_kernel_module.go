/*
 * Copyright 2018-2023 Arm Limited.
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

var (
	_          = pctx.StaticVariable("kmod_build", "${BobScriptsDir}/kmod_build.py")
	kbuildRule = pctx.StaticRule("kbuild",
		blueprint.RuleParams{
			Command: "python $kmod_build -o $out --depfile $depfile " +
				"--common-root ${SrcDir} " +
				"--module-dir $output_module_dir $extra_includes " +
				"--sources $in " +
				"--kernel $kernel_dir --cross-compile '$kernel_cross_compile' " +
				"$cc_flag $hostcc_flag $clang_triple_flag $ld_flag " +
				"$kbuild_options --extra-cflags='$extra_cflags' $make_args",
			CommandDeps: []string{"$kmod_build"},
			Depfile:     "$out.d",
			Deps:        blueprint.DepsGCC,
			Pool:        blueprint.Console,
			Description: "$out",
		}, "depfile", "extra_includes", "extra_cflags", "kernel_dir", "kernel_cross_compile",
		"kbuild_options", "make_args", "output_module_dir", "cc_flag", "hostcc_flag", "clang_triple_flag", "ld_flag")
)

func (g *linuxGenerator) kernelModOutputDir(m *kernelModule) string {
	return filepath.Join("${BuildDir}", "target", "kernel_modules", m.outputName())
}

func (g *linuxGenerator) kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext) {
	// Calculate and record outputs
	m.outputdir = g.kernelModOutputDir(m)
	m.outs = []string{filepath.Join(m.outputDir(), m.outputName()+".ko")}
	optional := !isBuiltByDefault(m)

	args := m.generateKbuildArgs(ctx).toDict()
	delete(args, "kmod_build")
	sources := utils.NewStringSlice(
		getBackendPathsInSourceDir(g, m.Properties.getSourcesResolved(ctx)),
		m.extraSymbolsFiles(ctx))

	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     kbuildRule,
			Outputs:  m.outputs(),
			Inputs:   sources,
			Optional: true,
			Args:     args,
		})

	// Add a dependency between Module.symvers and the kernel module. This
	// should really be added to Outputs or ImplicitOutputs above, but
	// Ninja doesn't support dependency files with multiple outputs yet.
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     blueprint.Phony,
			Inputs:   m.outputs(),
			Outputs:  []string{filepath.Join(m.outputDir(), "Module.symvers")},
			Optional: true,
		})

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, optional)
}
