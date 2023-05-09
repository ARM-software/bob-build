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
func (g *linuxGenerator) generateCommonActions(m *ModuleGenerateCommon, ctx blueprint.ModuleContext, inouts []inout) {
	m.outputdir = g.sourceOutputDir(ctx.Module())
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

func transformCmdAndroidToOld(cmd string, agr *ModuleGenrule) (retCmd *string) {
	// $(location) <label> -> ${tool} <label>
	// $(in) -> ${in}
	// $(out) -> ${out}
	// $(depfile) -> ${depfile}
	// $(genDir) -> ${gen_dir}
	newCmd := strings.Replace(cmd, "$(in)", "${in}", -1)
	newCmd = strings.Replace(newCmd, "$(out)", "${out}", -1)
	// We do not support $(locations)
	//newCmd = strings.Replace(newCmd, "$(locations)", "${tool}", -1)
	newCmd = strings.Replace(newCmd, "$(depfile)", "${depfile}", -1)
	newCmd = strings.Replace(newCmd, "$(genDir)", "${gen_dir}", -1)
	if strings.Contains(cmd, "$(location)") {
		toolFilesLength := len(agr.ModuleGenruleCommon.Properties.Tool_files)
		toolDepsLength := len(agr.ModuleGenruleCommon.Properties.Tools)
		if toolDepsLength >= 1 && toolFilesLength >= 1 {
			utils.Die("You cannot have default $(location) specified in Cmd if setting both tool_files and tools.")
		} else if toolDepsLength >= 1 {
			if strings.Contains(agr.ModuleGenruleCommon.Properties.Tools[0], ":") {
				newCmd = strings.Replace(newCmd, "$(location)", "$(location "+strings.Split(agr.ModuleGenruleCommon.Properties.Tools[0], ":")[0]+")", -1)
			} else {
				newCmd = strings.Replace(newCmd, "$(location)", "$(location "+agr.ModuleGenruleCommon.Properties.Tools[0]+")", -1)
			}
		} else {
			newCmd = strings.Replace(newCmd, "$(location)", "$(location "+agr.ModuleGenruleCommon.Properties.Tool_files[0]+")", -1)
		}
	}

	return &newCmd
}

func contains(s []string, elem string) bool {
	for _, v := range s {
		if elem == v {
			return true
		}
	}
	return false
}

func transformToolsAndroidToOld(gr *ModuleGenrule, gs *ModuleGenerateSource) {
	/*
		Bob handles multiple tool files identically to android. e.g.
		$(location tool2) == ${tool tool2}
		However, android differs as it also allows you to use the tag to depend
		on a tool created by a source dependency. Bob does this with special wildcards e.g.
		$(location dependency) == ${dependency_out}
		We must convert these correctly for the proxy object.
	*/
	// Extract each substr that is a 'location <tag>'
	matches := locationTagRegex.FindAllStringSubmatch(*gr.ModuleGenruleCommon.Properties.Cmd, -1)

	for _, v := range matches {
		tag := v[1]
		// If the tag refers to a tool inside of tool_files, we can just convert it the old command.
		if contains(gr.ModuleGenruleCommon.Properties.Tool_files, tag) {
			newString := strings.Replace(v[0], "$(location", "${tool", 1)
			newString = strings.Replace(newString, ")", "}", 1)
			newCmd := strings.Replace(*gr.ModuleGenruleCommon.Properties.Cmd, v[0], newString, 1)
			gr.ModuleGenruleCommon.Properties.Cmd = &newCmd
			continue
		}

		if tag[0] == ':' { // Tag is a dependency
			newString := strings.TrimPrefix(tag, ":")
			newString = "${" + newString + "_out}"
			newCmd := strings.Replace(*gr.ModuleGenruleCommon.Properties.Cmd, v[0], newString, 1)
			gr.ModuleGenruleCommon.Properties.Cmd = &newCmd
			continue
		}

		// If we arrive here, currently assume the tag is for a tool dependency from the 'tools' android attribute.
		// This should be the only other case, we leave it as the default with no check as the 'tools' attribute will not match
		// the tag. This is because it will specify a variant with <tag>:host. Let the linux generator expand this for us and cause an error
		// for now. On the linux backend we use generated_deps to support this, so add the correct prefix.
		newString := "${" + tag + "_out}"
		newCmd := strings.Replace(*gr.ModuleGenruleCommon.Properties.Cmd, v[0], newString, 1)
		gr.ModuleGenruleCommon.Properties.Cmd = &newCmd
	}
}

func (g *linuxGenerator) androidGenerateRuleActions(gr *ModuleGenrule, ctx blueprint.ModuleContext) {
	// TODO: remove proxy object and add a proper backend support.
	// If needed, refactor backend to accept both objects.
	// This approach is fragile, the generator runs after all the mutators have already executed and as such
	// we have to assume some properties may have been modified.

	// Re-use old Bob Code during transition by creating a proxy generateSource object to pass to the old generator
	var proxyGenerateSource ModuleGenerateSource
	proxyGenerateSource.SimpleName.Properties.Name = gr.ModuleGenruleCommon.Properties.Name
	gr.ModuleGenruleCommon.Properties.Cmd = transformCmdAndroidToOld(*gr.ModuleGenruleCommon.Properties.Cmd, gr)
	transformToolsAndroidToOld(gr, &proxyGenerateSource)
	proxyGenerateSource.ModuleGenerateCommon.Properties.Cmd = gr.ModuleGenruleCommon.Properties.Cmd
	proxyGenerateSource.ModuleGenerateCommon.Properties.Tools = gr.ModuleGenruleCommon.Properties.Tool_files
	proxyGenerateSource.ModuleGenerateCommon.Properties.Generated_deps = append(proxyGenerateSource.ModuleGenerateCommon.Properties.Generated_deps, gr.ModuleGenruleCommon.Properties.Tools...)
	proxyGenerateSource.ModuleGenerateCommon.Properties.Export_gen_include_dirs = gr.ModuleGenruleCommon.Properties.Export_include_dirs
	proxyGenerateSource.ModuleGenerateCommon.Properties.Srcs = gr.ModuleGenruleCommon.Properties.Srcs
	proxyGenerateSource.ModuleGenerateCommon.Properties.Exclude_srcs = gr.ModuleGenruleCommon.Properties.Exclude_srcs
	proxyGenerateSource.ModuleGenerateCommon.Properties.Depfile = gr.ModuleGenruleCommon.Properties.Depfile
	proxyGenerateSource.ModuleGenerateCommon.Properties.ResolveFiles(ctx, g)

	proxyGenerateSource.Properties.Implicit_srcs = gr.ModuleGenruleCommon.Properties.Tool_files
	proxyGenerateSource.Properties.Out = gr.Properties.Out
	proxyGenerateSource.ResolveFiles(ctx, g)

	g.generateSourceActions(&proxyGenerateSource, ctx)

	// This is the generated paths for the outs, needed to correctly depend upon these rules
	gr.ModuleGenruleCommon.outs = proxyGenerateSource.ModuleGenerateCommon.outs
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

	tocFile := g.getSharedLibTocPath(m)
	g.addSharedLibToc(ctx, soFile, tocFile, m.getTarget())

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
			Rule:     copyRule,
			Inputs:   m.outputs(),
			Outputs:  []string{g.getBinaryPath(m)},
			Optional: true,
		})

	installDeps := append(g.install(m, ctx), g.getPhonyFiles(m)...)
	addPhony(m, ctx, installDeps, !isBuiltByDefault(m))
}
