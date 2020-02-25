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
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/bpwriter"
)

func (g *androidBpGenerator) genBinaryActions(m *generateBinary, mctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		panic(fmt.Errorf("Generated binaries are not supported (%s)", m.Name()))
	}
}

func (g *androidBpGenerator) genSharedActions(m *generateSharedLibrary, mctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		panic(fmt.Errorf("Generated shared libraries are not supported (%s)", m.Name()))
	}
}

func (g *androidBpGenerator) genStaticActions(m *generateStaticLibrary, mctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		panic(fmt.Errorf("Generated static libraries are not supported (%s)", m.Name()))
	}
}

func populateCommonProps(gc *generateCommon, mctx blueprint.ModuleContext, m bpwriter.Module) {
	// Replace ${args} immediately
	cmd := strings.Replace(proptools.String(gc.Properties.Cmd), "${args}",
		strings.Join(gc.Properties.Args, " "), -1)
	m.AddString("cmd", cmd)

	if gc.Properties.Tool != nil {
		m.AddString("tool", *gc.Properties.Tool)
	}
	if gc.Properties.Rsp_content != nil {
		m.AddString("rsp_content", *gc.Properties.Rsp_content)
	}
	if gc.Properties.Host_bin != nil {
		m.AddString("host_bin", ccModuleName(mctx, gc.getHostBinModule(mctx).Name()))
	}
	m.AddBool("depfile", proptools.Bool(gc.Properties.Depfile))

	m.AddStringList("module_deps", gc.Properties.Module_deps)
	m.AddStringList("module_srcs", gc.Properties.Module_srcs)
	m.AddStringList("encapsulates", gc.Properties.Encapsulates)
	m.AddStringList("export_gen_include_dirs", gc.Properties.Export_gen_include_dirs)
	m.AddStringList("cflags", gc.Properties.FlagArgsBuild.Cflags)
	m.AddStringList("conlyflags", gc.Properties.FlagArgsBuild.Conlyflags)
	m.AddStringList("cxxflags", gc.Properties.FlagArgsBuild.Cxxflags)
	m.AddStringList("asflags", gc.Properties.FlagArgsBuild.Asflags)
	m.AddStringList("ldflags", gc.Properties.FlagArgsBuild.Ldflags)
	m.AddStringList("ldlibs", gc.Properties.FlagArgsBuild.Ldlibs)
}

func (g *androidBpGenerator) generateSourceActions(gs *generateSource, mctx blueprint.ModuleContext, inouts []inout) {
	if !enabledAndRequired(gs) {
		return
	}
	// Calculate and record outputs
	gs.recordOutputsFromInout(inouts)

	m, err := AndroidBpFile().NewModule("genrule_bob", gs.shortName())
	if err != nil {
		panic(err.Error())
	}

	m.AddStringList("srcs", gs.generateCommon.Properties.getSources(mctx))
	m.AddStringList("out", gs.Properties.Out)
	m.AddStringList("implicit_srcs", gs.Properties.Implicit_srcs)
	m.AddStringList("implicit_outs", gs.Properties.Implicit_outs)

	populateCommonProps(&gs.generateCommon, mctx, m)
}

func (g *androidBpGenerator) transformSourceActions(ts *transformSource, mctx blueprint.ModuleContext, inouts []inout) {
	if !enabledAndRequired(ts) {
		return
	}
	// Calculate and record outputs
	ts.recordOutputsFromInout(inouts)

	m, err := AndroidBpFile().NewModule("genrule_bob", ts.shortName())
	if err != nil {
		panic(err.Error())
	}

	m.AddStringList("multi_out_srcs", ts.generateCommon.Properties.getSources(mctx))

	gr := m.NewGroup("multi_out_props")
	// if REs had double slashes in original value, at parsing they got removed, so compensate for that
	gr.AddString("match", strings.Replace(ts.Properties.TransformSourceProps.Out.Match, "\\", "\\\\", -1))
	gr.AddStringList("replace", ts.Properties.TransformSourceProps.Out.Replace)
	gr.AddStringList("implicit_srcs", ts.Properties.TransformSourceProps.Out.Implicit_srcs)
	gr.AddStringList("implicit_outs", ts.Properties.TransformSourceProps.Out.Implicit_outs)

	populateCommonProps(&ts.generateCommon, mctx, m)
}
