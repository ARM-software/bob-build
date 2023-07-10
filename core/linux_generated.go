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
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
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

type ruleContext struct {
	rule      *blueprint.Rule
	inouts    []inout
	args      map[string]string
	implicits []string
}

func (g *linuxGenerator) buildRules(r *ruleContext, ctx blueprint.ModuleContext) {

	for _, inout := range r.inouts {
		r.args["_out_"] = utils.Join(inout.out)
		if inout.rspfile != "" {
			r.args["rspfile"] = inout.rspfile
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

		unique_implicits := utils.Unique(append(inout.implicitSrcs, r.implicits...))

		buildparams := blueprint.BuildParams{
			Rule:            *r.rule,
			Inputs:          inout.in,
			Outputs:         mainRuleOuts,
			ImplicitOutputs: mainRuleImplicitOuts,
			Implicits:       unique_implicits,
			Args:            r.args,
			Optional:        true,
			Depfile:         inout.depfile,
			Deps:            deps,
		}

		ctx.Build(pctx, buildparams)
	}
}

// Generate the build actions for a generateSource module and populates the outputs.
func (g *linuxGenerator) generateCommonActions(m *ModuleGenerateCommon, ctx blueprint.ModuleContext, inouts []inout) {
	outputdir := backend.Get().SourceOutputDir(ctx.Module())
	prefixInoutsWithOutputDir(inouts, outputdir)

	cmd, args, implicits, hostTarget := m.getArgs(ctx)
	ldLibraryPath := ""
	if _, ok := args["host_bin"]; ok {
		ldLibraryPath += "LD_LIBRARY_PATH=" + backend.Get().SharedLibsDir(hostTarget) + ":$$LD_LIBRARY_PATH "
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

	rule := ctx.Rule(pctx, "gen_"+m.Name(), ruleparams,
		append(utils.SortedKeys(args), "depfile", "rspfile", "_out_")...)

	rCtx := ruleContext{
		rule:      &rule,
		inouts:    inouts,
		args:      args,
		implicits: implicits,
	}

	g.buildRules(&rCtx, ctx)
}

// Generate the build actions for a `ModuleGenrule` & `ModuleGensrcs` module and populates the outputs.
func (g *linuxGenerator) generateStrictCommonActions(m *ModuleStrictGenerateCommon, ctx blueprint.ModuleContext, inouts []inout) {
	outputdir := backend.Get().SourceOutputDir(ctx.Module())
	prefixInoutsWithOutputDir(inouts, outputdir)

	cmd, args, implicits, hostLdLibraryPath := m.getArgs(ctx)

	var pool blueprint.Pool
	ruleparams := blueprint.RuleParams{
		Command: hostLdLibraryPath + cmd,
		// Restat is always set to true. This is due to wanting to enable scripts
		// to only update the outputs if they have changed (keeping the same mtime if it
		// has not). If there are no updates, the following rules will not have to update
		// the output.
		Restat:      true,
		Pool:        pool,
		Description: "$out",
	}

	rule := ctx.Rule(pctx, "gen_"+m.Name(), ruleparams,
		append(utils.SortedKeys(args), "depfile", "_out_")...)

	rCtx := ruleContext{
		rule:      &rule,
		inouts:    inouts,
		args:      args,
		implicits: implicits,
	}

	g.buildRules(&rCtx, ctx)
}

func (g *linuxGenerator) genruleActions(gr *ModuleGenrule, ctx blueprint.ModuleContext) {
	inouts := gr.generateInouts(ctx)
	g.generateStrictCommonActions(&gr.ModuleStrictGenerateCommon, ctx, inouts)

	addPhony(gr, ctx, g.getPhonyFiles(gr), !isBuiltByDefault(gr))
}

func (g *linuxGenerator) gensrcsActions(gr *ModuleGensrcs, ctx blueprint.ModuleContext) {
	inouts := gr.generateInouts(ctx)
	g.generateStrictCommonActions(&gr.ModuleStrictGenerateCommon, ctx, inouts)

	addPhony(gr, ctx, g.getPhonyFiles(gr), !isBuiltByDefault(gr))
}

func (g *linuxGenerator) generateSourceActions(m *ModuleGenerateSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)

	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) transformSourceActions(m *ModuleTransformSource, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

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

	if toc, ok := m.OutFiles().FindSingle(
		func(p file.Path) bool { return p.IsType(file.TypeToc) }); ok {
		g.addSharedLibToc(ctx, soFile, toc.BuildPath(), m.getTarget())
	}

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}

func (g *linuxGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext) {
	inouts := m.generateInouts(ctx, g)
	g.generateCommonActions(&m.ModuleGenerateCommon, ctx, inouts)

	// Create a rule to copy the generated binary
	// from gen_dir to the common binary directory
	ctx.Build(pctx,
		blueprint.BuildParams{
			Rule:   copyRule,
			Inputs: m.outputs(),
			Outputs: []string{filepath.Join(
				backend.Get().BinaryOutputDir(m.getTarget()),
				m.outputFileName())},
			Optional: true,
		})

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}
