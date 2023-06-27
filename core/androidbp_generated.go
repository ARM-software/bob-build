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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/utils"
)

var dependencyRegex = regexp.MustCompile(`\$\{([a-zA-Z0-9\/\.:_-]+)_out\}`)

func (g *androidBpGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		utils.Die("Generated binaries are not supported (%s)", m.Name())
	}
}

func (g *androidBpGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		utils.Die("Generated shared libraries are not supported (%s)", m.Name())
	}
}

func (g *androidBpGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		utils.Die("Generated static libraries are not supported (%s)", m.Name())
	}
}

func expandCmd(gc *ModuleGenerateCommon, ctx blueprint.ModuleContext, s string) string {
	return utils.Expand(s, func(s string) string {
		switch s {
		case "src_dir":
			// All modules are written to the same Android.bp, at the project root,
			// so Bob's `src_dir` (i.e. the project root) just maps to module dir.
			return "${module_dir}"
		case "module_dir":
			// ...whereas module_dir refers to the directory containing the
			// build.bp - so we need to expand it before it's "flattened" into a
			// single Android.bp file. Also prefix with the directory containing
			// the Android.bp, which makes the result relative to the working
			// directory (= the root of the Android tree). This is required because
			// the result will be used directly in `cmd`, rather than being
			// included in a `srcs` field which would be processed further.
			return filepath.Join("${module_dir}", ctx.ModuleDir())
		case "bob_config":
			if !proptools.Bool(gc.Properties.Depfile) {
				utils.Die("%s references Bob config but depfile not enabled. "+
					"Config dependencies must be declared via a depfile!", gc.Name())
			}
			return config.GetEnvironmentVariables().ConfigFile
		case "bob_config_json":
			if !proptools.Bool(gc.Properties.Depfile) {
				utils.Die("%s references Bob config but depfile not enabled. "+
					"Config dependencies must be declared via a depfile!", gc.Name())
			}
			return config.GetEnvironmentVariables().ConfigJSON
		case "bob_config_opts":
			return config.GetEnvironmentVariables().ConfigOpts
		default:
			if strings.HasPrefix(s, "tool ") {
				toolPath := strings.TrimSpace(strings.TrimPrefix(s, "tool "))
				return "${tool " + toolPath + "}"
			}
			return "${" + s + "}"
		}
	})
}

func populateCommonProps(gc *ModuleGenerateCommon, ctx blueprint.ModuleContext, m bpwriter.Module) {
	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(gc.Properties.Cmd), "${args}",
		strings.Join(gc.Properties.Args, " "), -1)
	cmd = expandCmd(gc, ctx, cmd)
	m.AddString("cmd", cmd)

	if len(gc.Properties.Tools) > 0 {
		m.AddStringList("tools", gc.Properties.Tools)
	}
	if gc.Properties.Rsp_content != nil {
		m.AddString("rsp_content", *gc.Properties.Rsp_content)
	}
	if gc.Properties.Host_bin != nil {
		hostBin := bpModuleNamesForDep(ctx, gc.hostBinName(ctx))
		if len(hostBin) != 1 {
			utils.Die("%s must have one host_bin entry (have %d)", gc.Name(), len(hostBin))
		}
		m.AddString("host_bin", hostBin[0])
	}
	if proptools.Bool(gc.Properties.Depfile) && !utils.ContainsArg(cmd, "depfile") {
		utils.Die("%s depfile is true, but ${depfile} not used in cmd", gc.Name())
	}

	m.AddBool("depfile", proptools.Bool(gc.Properties.Depfile))

	m.AddStringList("generated_deps", getShortNamesForDirectDepsWithTagsForNonFilegroup(ctx, GeneratedTag))
	m.AddStringList("generated_sources", getShortNamesForDirectDepsWithTags(ctx, GeneratedSourcesTag))
	m.AddStringList("export_gen_include_dirs", gc.Properties.Export_gen_include_dirs)
	m.AddStringList("cflags", gc.Properties.FlagArgsBuild.Cflags)
	m.AddStringList("conlyflags", gc.Properties.FlagArgsBuild.Conlyflags)
	m.AddStringList("cxxflags", gc.Properties.FlagArgsBuild.Cxxflags)
	m.AddStringList("asflags", gc.Properties.FlagArgsBuild.Asflags)
	m.AddStringList("ldflags", gc.Properties.FlagArgsBuild.Ldflags)
	m.AddStringList("ldlibs", gc.Properties.FlagArgsBuild.Ldlibs)
}

func expandGenruleCmd(gc *ModuleGenruleCommon, ctx blueprint.ModuleContext, s string) string {
	return utils.Expand(s, func(s string) string {
		switch s {
		case "src_dir":
			// TODO: Implement me
			return "$(" + s + ")"
		case "module_dir":
			// TODO: Implement me
			return "$(" + s + ")"
		case "gen_dir":
			return "$(genDir)"
		case "bob_config":
			if !proptools.Bool(gc.Properties.Depfile) {
				utils.Die("%s references Bob config but depfile not enabled. "+
					"Config dependencies must be declared via a depfile!", gc.Name())
			}
			return config.GetEnvironmentVariables().ConfigFile
		case "bob_config_json":
			if !proptools.Bool(gc.Properties.Depfile) {
				utils.Die("%s references Bob config but depfile not enabled. "+
					"Config dependencies must be declared via a depfile!", gc.Name())
			}
			return config.GetEnvironmentVariables().ConfigJSON
		case "bob_config_opts":
			return config.GetEnvironmentVariables().ConfigOpts
		default:
			return "$(" + s + ")"
		}
	})
}

func (g *androidBpGenerator) androidGenerateCommonActions(gc *ModuleGenruleCommon, ctx blueprint.ModuleContext, m bpwriter.Module) {
	m.AddStringList("srcs", gc.Properties.Srcs)
	m.AddStringList("exclude_srcs", gc.Properties.Exclude_srcs)
	// `Cmd` has to be parsed back from ${(name)_out} to $(location name)
	changeCmdToolFilesToLocation(gc)

	// expand special variables
	cmd := expandGenruleCmd(gc, ctx, *gc.Properties.Cmd)

	m.AddString("cmd", strings.TrimSpace(cmd))
	m.AddOptionalBool("depfile", gc.Properties.Depfile)
	m.AddOptionalBool("enabled", gc.Properties.Enabled)
	m.AddStringList("export_include_dirs", gc.Properties.Export_include_dirs)
	m.AddStringList("tool_files", gc.Properties.Tool_files)

	var tools []string = []string{}

	for _, t := range gc.Properties.Tools {
		if strings.HasSuffix(t, ":host") || strings.HasSuffix(t, ":target") {
			idx := strings.LastIndex(t, ":")
			tgt := t[idx+1:]

			hostBinModule := ctx.GetDirectDepWithTag(t[:idx], HostToolBinaryTag)

			if hostBinModule != nil {
				if s, ok := hostBinModule.(splittable); ok {
					variants := s.supportedVariants()
					if len(variants) > 1 {
						tools = append(tools, hostBinModule.Name()+"__"+tgt)
					} else {
						tools = append(tools, t[:idx])
					}
				} else {
					panic(fmt.Errorf("'%s' is not a host tool", t))
				}
			}
		} else {
			tools = append(tools, t)
		}
	}

	m.AddStringList("tools", tools)
}

func changeCmdToolFilesToLocation(gc *ModuleGenruleCommon) {

	tool_files_targets := utils.MixedListToBobTargets(gc.Properties.AndroidGenerateCommonProps.Tool_files)

	// find all `${moduleName_out}`
	matches := dependencyRegex.FindAllStringSubmatch(*gc.Properties.AndroidGenerateCommonProps.Cmd, -1)

	for _, v := range matches {
		tag := v[1]

		if utils.Contains(tool_files_targets, tag) {
			newString := fmt.Sprintf("$(location :%s)", tag)
			newCmd := strings.Replace(*gc.Properties.AndroidGenerateCommonProps.Cmd, v[0], newString, 1)
			gc.Properties.AndroidGenerateCommonProps.Cmd = &newCmd
		}
	}
}

func (g *androidBpGenerator) genruleActions(gr *ModuleGenrule, ctx blueprint.ModuleContext) {
	m, err := AndroidBpFile().NewModule("genrule", gr.shortName())
	if err != nil {
		utils.Die("%v", err.Error())
	}
	g.androidGenerateCommonActions(&gr.ModuleGenruleCommon, ctx, m)
	m.AddStringList("out", gr.Properties.Out)
}

func (g *androidBpGenerator) gensrcsActions(gr *ModuleGensrcs, ctx blueprint.ModuleContext) {
	m, err := AndroidBpFile().NewModule("gensrcs", gr.shortName())
	if err != nil {
		utils.Die("%v", err.Error())
	}
	g.androidGenerateCommonActions(&gr.ModuleGenruleCommon, ctx, m)
	m.AddString("output_extension", gr.Properties.Output_extension)
}

func (g *androidBpGenerator) generateSourceActions(gs *ModuleGenerateSource, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(gs) {
		return
	}

	m, err := AndroidBpFile().NewModule("genrule_bob", gs.shortName())
	if err != nil {
		utils.Die("%v", err.Error())
	}

	srcs := []string{}
	implicits := []string{}

	gs.ModuleGenerateCommon.Properties.GetDirectFiles().ForEach(
		func(fp file.Path) bool {

			if fp.IsType(file.TypeImplicit) {
				implicits = append(implicits, fp.UnScopedPath())
			} else {
				srcs = append(srcs, fp.UnScopedPath())
			}

			return true
		})

	// Add `filegroup_srcs` and ':module_name' source dependencies.
	// Both has to be in colon notation (`:module_name`)
	// as `genrule_bob` does not support `filegroup_srcs`.
	srcs = append(srcs, utils.PrefixAll(utils.MixedListToBobTargets(gs.ModuleGenerateCommon.Properties.Srcs), ":")...)
	srcs = append(srcs, utils.PrefixAll(gs.ModuleGenerateCommon.Properties.Filegroup_srcs, ":")...)

	m.AddStringList("srcs", srcs)
	m.AddStringList("out", gs.Properties.Out)
	m.AddStringList("implicit_srcs", implicits)

	populateCommonProps(&gs.ModuleGenerateCommon, ctx, m)

	// No AndroidProps in gen sources, so always in vendor for now
	addInstallProps(m, gs.getInstallableProps(), true)
}

func (g *androidBpGenerator) transformSourceActions(ts *ModuleTransformSource, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(ts) {
		return
	}

	m, err := AndroidBpFile().NewModule("gensrcs_bob", ts.shortName())
	if err != nil {
		utils.Die(err.Error())
	}

	srcs := []string{}
	ts.ModuleGenerateCommon.Properties.GetDirectFiles().ForEach(
		func(fp file.Path) bool {
			srcs = append(srcs, fp.UnScopedPath())
			return true
		})
	m.AddStringList("srcs", srcs)

	gr := m.NewGroup("out")
	// if REs had double slashes in original value, at parsing they got removed, so compensate for that
	gr.AddString("match", strings.Replace(ts.Properties.TransformSourceProps.Out.Match, "\\", "\\\\", -1))
	gr.AddStringList("replace", ts.Properties.TransformSourceProps.Out.Replace)
	gr.AddStringList("implicit_srcs", ts.Properties.TransformSourceProps.Out.Implicit_srcs)

	populateCommonProps(&ts.ModuleGenerateCommon, ctx, m)

	// No AndroidProps in gen sources, so always in vendor for now
	addInstallProps(m, ts.getInstallableProps(), true)
}
