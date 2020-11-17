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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/utils"
)

func (g *androidBpGenerator) genBinaryActions(m *generateBinary, mctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		panic(fmt.Errorf("Generated binaries are not supported (%s)", m.Name()))
	}
}

func (g *androidBpGenerator) genSharedActions(m *generateSharedLibrary, mctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		panic(fmt.Errorf("Generated shared libraries are not supported (%s)", m.Name()))
	}
}

func (g *androidBpGenerator) genStaticActions(m *generateStaticLibrary, mctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		panic(fmt.Errorf("Generated static libraries are not supported (%s)", m.Name()))
	}
}

func expandCmd(s string, moduleDir string) string {
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
			return filepath.Join("${module_dir}", moduleDir)
		case "bob_config":
			return configFile
		case "bob_config_json":
			return configJSONFile
		case "bob_config_opts":
			return configOpts
		default:
			return "${" + s + "}"
		}
	})
}

func populateCommonProps(gc *generateCommon, mctx blueprint.ModuleContext, m bpwriter.Module) {
	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(gc.Properties.Cmd), "${args}",
		strings.Join(gc.Properties.Args, " "), -1)
	cmd = expandCmd(cmd, mctx.ModuleDir())
	m.AddString("cmd", cmd)

	if gc.Properties.Tool != nil {
		m.AddString("tool", *gc.Properties.Tool)
	}
	if gc.Properties.Rsp_content != nil {
		m.AddString("rsp_content", *gc.Properties.Rsp_content)
	}
	if gc.Properties.Host_bin != nil {
		m.AddString("host_bin", ccModuleName(mctx, gc.hostBinName(mctx)))
	}
	m.AddBool("depfile", proptools.Bool(gc.Properties.Depfile))

	m.AddStringList("module_deps", getShortNamesForDirectDepsWithTags(mctx, generatedDepTag))
	m.AddStringList("module_srcs", getShortNamesForDirectDepsWithTags(mctx, generatedSourceTag))
	m.AddStringList("encapsulates", gc.Properties.Encapsulates)
	m.AddStringList("export_gen_include_dirs", gc.Properties.Export_gen_include_dirs)
	m.AddStringList("cflags", gc.Properties.FlagArgsBuild.Cflags)
	m.AddStringList("conlyflags", gc.Properties.FlagArgsBuild.Conlyflags)
	m.AddStringList("cxxflags", gc.Properties.FlagArgsBuild.Cxxflags)
	m.AddStringList("asflags", gc.Properties.FlagArgsBuild.Asflags)
	m.AddStringList("ldflags", gc.Properties.FlagArgsBuild.Ldflags)
	m.AddStringList("ldlibs", gc.Properties.FlagArgsBuild.Ldlibs)
}

func (g *androidBpGenerator) generateSourceActions(gs *generateSource, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(gs) {
		return
	}

	m, err := AndroidBpFile().NewModule("genrule_bob", gs.shortName())
	if err != nil {
		panic(err.Error())
	}

	m.AddStringList("srcs", gs.generateCommon.Properties.getSources(mctx))
	m.AddStringList("out", gs.Properties.Out)
	m.AddStringList("implicit_srcs", gs.Properties.getImplicitSources(mctx))
	m.AddStringList("implicit_outs", gs.Properties.Implicit_outs)

	populateCommonProps(&gs.generateCommon, mctx, m)

	// No AndroidProps in gen sources, so always in vendor for now
	addInstallProps(m, gs.getInstallableProps(), true)
}

func (g *androidBpGenerator) transformSourceActions(ts *transformSource, mctx blueprint.ModuleContext) {
	if !enabledAndRequired(ts) {
		return
	}

	m, err := AndroidBpFile().NewModule("gensrcs_bob", ts.shortName())
	if err != nil {
		panic(err.Error())
	}

	m.AddStringList("srcs", ts.generateCommon.Properties.getSources(mctx))
	gr := m.NewGroup("out")
	// if REs had double slashes in original value, at parsing they got removed, so compensate for that
	gr.AddString("match", strings.Replace(ts.Properties.TransformSourceProps.Out.Match, "\\", "\\\\", -1))
	gr.AddStringList("replace", ts.Properties.TransformSourceProps.Out.Replace)
	gr.AddStringList("implicit_srcs", ts.Properties.TransformSourceProps.Out.Implicit_srcs)
	gr.AddStringList("implicit_outs", ts.Properties.TransformSourceProps.Out.Implicit_outs)

	populateCommonProps(&ts.generateCommon, mctx, m)

	// No AndroidProps in gen sources, so always in vendor for now
	addInstallProps(m, ts.getInstallableProps(), true)
}
