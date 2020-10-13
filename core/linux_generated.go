/*
 * Copyright 2018-2020 Arm Limited.
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

	"github.com/ARM-software/bob-build/internal/utils"
)

var copyRule = pctx.StaticRule("copy",
	blueprint.RuleParams{
		Command:     "cp $in $out",
		Description: "$out",
	})

var touchRule = pctx.StaticRule("touch",
	blueprint.RuleParams{
		Command:     "touch -c $out",
		Description: "touch $out",
	})

// Generate the build actions for a generateSource module and populates the outputs.
func (g *linuxGenerator) generateCommonActions(m *generateCommon, ctx blueprint.ModuleContext, inouts []inout) {
	m.outputdir = g.sourceOutputDir(m)
	prefixInoutsWithOutputDir(inouts, m.outputDir())
	// Calculate and record outputs and include dirs
	m.recordOutputsFromInout(inouts)
	m.includeDirs = utils.PrefixDirs(m.Properties.Export_gen_include_dirs, m.outputDir())
	m.encapsulatedOuts = getGeneratedEncapsulatedFiles(ctx)

	cmd, args, implicits, hostTarget := m.getArgs(ctx)

	ldLibraryPath := ""
	if _, ok := args["host_bin"]; ok {
		ldLibraryPath += "LD_LIBRARY_PATH=" + filepath.Join("${BuildDir}", string(hostTarget), "shared") + ":$$LD_LIBRARY_PATH "
	}
	utils.StripUnusedArgs(args, cmd)

	var pool blueprint.Pool
	if m.Properties.Console {
		// Console can be used to run longrunning jobs (even interactive jobs).
		pool = blueprint.Console
	}

	ruleparams := blueprint.RuleParams{
		Command: ldLibraryPath + cmd,
		// Restat is always set to true. This is due to wanting to enable scripts
		// to only update the outputs if they have changed (keeping the same mtime if it
		// has not). If there are no updates, the following rules will not have to update
		// the output.
		Restat:      true,
		Pool:        pool,
		Description: "$out",
	}

	if m.Properties.Rsp_content != nil {
		ruleparams.Rspfile = "${rspfile}"
		ruleparams.RspfileContent = *m.Properties.Rsp_content
	}

	//print("Keys:" + strings.Join(argkeys, ",") + "\n")
	rule := ctx.Rule(pctx, "gen_"+m.Name(), ruleparams,
		append(utils.SortedKeys(args), "depfile", "rspfile")...)

	for _, inout := range inouts {
		if inout.depfile != "" && len(inout.out) > 1 {
			panic(fmt.Errorf("Module %s uses a depfile with multiple outputs", ctx.ModuleName()))
		}

		if inout.rspfile != "" {
			args["rspfile"] = inout.rspfile
		}

		if _, ok := args["headers_generated"]; ok {
			headers := utils.Filter(utils.IsHeader, inout.out)
			args["headers_generated"] = strings.Join(headers, " ")
		}
		if _, ok := args["srcs_generated"]; ok {
			sources := utils.Filter(utils.IsNotHeader, inout.out)
			args["srcs_generated"] = strings.Join(sources, " ")
		}

		buildparams := blueprint.BuildParams{
			Rule:      rule,
			Inputs:    inout.in,
			Outputs:   inout.out,
			Implicits: append(inout.implicitSrcs, implicits...),
			Args:      args,
			Optional:  true,
		}

		// ninja currently does not support case when depfile is defined and
		// multiple outputs at the same time. For implicit outputs fallback to using a separate rule.
		if inout.depfile != "" {
			if len(inout.implicitOuts) > 0 {
				// No-op rule linking implicit outputs to the main output. Touch the implicit
				// outputs in case the script actually creates the implicit outputs first.
				ctx.Build(pctx,
					blueprint.BuildParams{
						Rule:     touchRule,
						Inputs:   inout.out,
						Outputs:  inout.implicitOuts,
						Optional: true,
					})
			}
			buildparams.Depfile = inout.depfile
			buildparams.Deps = blueprint.DepsGCC
		} else {
			buildparams.ImplicitOutputs = inout.implicitOuts
		}

		ctx.Build(pctx, buildparams)
	}
}

func (g *linuxGenerator) generateSourceActions(m *generateSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) transformSourceActions(m *transformSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	// Create a rule to copy the generated library
	// from gen_dir to the common library directory
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     copyRule,
			Inputs:   m.outputs(),
			Outputs:  []string{getSharedLibLinkPath(m)},
			Optional: true,
		})

	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	// Create a rule to copy the generated binary
	// from gen_dir to the common binary directory
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     copyRule,
			Inputs:   m.outputs(),
			Outputs:  []string{getBinaryPath(m)},
			Optional: true,
		})

	installDeps := g.install(m, ctx)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}
