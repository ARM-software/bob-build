/*
 * Copyright 2018-2022 Arm Limited.
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
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

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

	cmd, args, implicits, hostTarget := m.getArgs(ctx)

	ldLibraryPath := ""
	if _, ok := args["host_bin"]; ok {
		ldLibraryPath += "LD_LIBRARY_PATH=" + g.sharedLibsDir(hostTarget) + ":$$LD_LIBRARY_PATH "
	}
	utils.StripUnusedArgs(args, cmd)

	var pool blueprint.Pool
	if proptools.Bool(m.Properties.Console) {
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
		append(utils.SortedKeys(args), "depfile", "rspfile", "_out_")...)

	for _, inout := range inouts {
		args["_out_"] = utils.Join(inout.out)
		if inout.rspfile != "" {
			args["rspfile"] = inout.rspfile
		}

		mainRuleOuts := inout.out
		mainRuleImplicitOuts := inout.implicitOuts
		deps := blueprint.DepsNone

		// ninja currently does not support the case when depfile is
		// defined and multiple outputs at the same time. So adjust the
		// main rule to have a single output, and link the remaining
		// outputs using a separate rule.
		if inout.depfile != "" && (len(inout.out)+len(inout.implicitOuts)) > 1 {
			// No-op rule linking the extra outputs to the main
			// output. Update the extra outputs' mtime in case the
			// script actually creates the extra outputs first.

			allOutputs := append(mainRuleOuts, mainRuleImplicitOuts...)
			mainRuleOuts = allOutputs[0:1]
			mainRuleImplicitOuts = []string{}

			ctx.Build(pctx,
				blueprint.BuildParams{
					Rule:     touchRule,
					Inputs:   mainRuleOuts,
					Outputs:  allOutputs[1:],
					Optional: true,
				})
			deps = blueprint.DepsGCC
		}

		buildparams := blueprint.BuildParams{
			Rule:            rule,
			Inputs:          inout.in,
			Outputs:         mainRuleOuts,
			ImplicitOutputs: mainRuleImplicitOuts,
			Implicits:       append(inout.implicitSrcs, implicits...),
			Args:            args,
			Optional:        true,
			Depfile:         inout.depfile,
			Deps:            deps,
		}

		ctx.Build(pctx, buildparams)
	}
}

func transformCmdAndroidToOld(cmd string) (retCmd *string) {
	// $(location) <label> -> ${tool} <label>
	// $(in) -> ${in}
	// $(out) -> ${out}
	// $(depfile) -> ${depfile}
	// $(genDir) -> ${gen_dir}
	newCmd := strings.Replace(cmd, "$(in)", "${in}", -1)
	newCmd = strings.Replace(newCmd, "$(out)", "${out}", -1)
	newCmd = strings.Replace(newCmd, "$(locations)", "${tool}", -1)
	newCmd = strings.Replace(newCmd, "$(location)", "${tool}", -1)
	newCmd = strings.Replace(newCmd, "$(depfile)", "${depfile}", -1)
	newCmd = strings.Replace(newCmd, "$(genDir)", "${gen_dir}", -1)
	return &newCmd
}

func (g *linuxGenerator) androidGenerateRuleActions(ag *androidGenerateRule, mctx blueprint.ModuleContext) {

	// Re-use old Bob Code during transition by creating a proxy generateSource object to pass to the old generator
	var proxyGenerateSource generateSource
	proxyGenerateSource.SimpleName.Properties.Name = ag.androidGenerateCommon.Properties.Name
	proxyGenerateSource.generateCommon.Properties.Cmd = transformCmdAndroidToOld(*ag.androidGenerateCommon.Properties.Cmd)
	proxyGenerateSource.generateCommon.Properties.Tools = ag.androidGenerateCommon.Properties.Tool_files
	proxyGenerateSource.generateCommon.Properties.Export_gen_include_dirs = ag.androidGenerateCommon.Properties.Export_include_dirs
	proxyGenerateSource.generateCommon.Properties.Srcs = ag.androidGenerateCommon.Properties.Srcs
	proxyGenerateSource.generateCommon.Properties.Exclude_srcs = ag.androidGenerateCommon.Properties.Exclude_srcs
	proxyGenerateSource.generateCommon.Properties.Depfile = ag.androidGenerateCommon.Properties.Depfile
	proxyGenerateSource.Properties.Implicit_srcs = ag.androidGenerateCommon.Properties.Tool_files
	proxyGenerateSource.Properties.Out = ag.Properties.Out

	// Adds the correct prefixes to all paths for the linux backend
	proxyGenerateSource.processPaths(mctx, g)
	g.generateSourceActions(&proxyGenerateSource, mctx)

	// This is the generated paths for the outs, needed to correctly depend upon these rules
	ag.androidGenerateCommon.outs = proxyGenerateSource.generateCommon.outs
}

func (g *linuxGenerator) generateSourceActions(m *generateSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) transformSourceActions(m *transformSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.generateCommon, ctx, inouts)

	// Create a rule to copy the generated library
	// from gen_dir to the common library directory
	soFile := g.getSharedLibLinkPath(m)
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:     copyRule,
			Inputs:   m.outputs(),
			Outputs:  []string{soFile},
			Optional: true,
		})

	tocFile := g.getSharedLibTocPath(m)
	g.addSharedLibToc(ctx, soFile, tocFile, m.getTarget())

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
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
			Outputs:  []string{g.getBinaryPath(m)},
			Optional: true,
		})

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}
