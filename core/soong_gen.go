// +build soong

/*
 * Copyright 2019 Arm Limited.
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
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"android/soong/android"
	"android/soong/genrule"

	"github.com/google/blueprint/proptools"
)

func expandBobVariables(str, tool, hostBin string) (out string, err error) {
	// Bob is lax on whether there is any parenthesis, or whether
	// () or {} is used. That's because it relies on the expansion
	// happening in ninja. In a few cases Bob explicitly looks for
	// {}. Soong wants ()
	//
	// ${host_bin} => $(location tool) or $(location)
	// ${tool} => $(location tool_file) or $(location)
	// ${in} => $(in)
	// ${out} => $(out)
	// ${depfile} => $(depfile)
	// ${gen_dir} => $(genDir)
	//
	// ${bob_config} - expand inline
	// ${bob_config_opts} - expand inline
	// ${args} - expand inline
	//
	// {{match_srcs x}} => $(location x) ignoring globs
	//
	// ${src_dir}, ${module_dir}, ${xxmod_dir}, ${xxmod_outs} don't appear to be supported
	// Nor are ${srcs_generated}, ${headers_generated}
	//
	// flag_defaults is primarily used to invoke sub-makes of
	// different libraries. This shouldn't be needed on Android.
	// This means the following can't be expanded:
	//
	// ${ar}
	// ${as} ${asflags}
	// ${cc} ${cflags} ${conlyflags}
	// ${cxx} ${cxxflags}
	// ${linker} ${ldflags}
	out = os.Expand(str, func(s string) string {
		switch s {
		case "host_bin":
			if hostBin != "" {
				return "$(location " + hostBin + ")"
			} else {
				err = errors.New("${host_bin} used but host_bin not specified")
				return "$(location)"
			}
		case "tool":
			if tool != "" {
				return "$(location " + tool + ")"
			} else {
				err = errors.New("${tool} used but tool not specified")
				return "$(location)"
			}
		case "in":
			return "$(in)"
		case "out":
			return "$(out)"
		case "depfile":
			return "$(depfile)"
		case "gen_dir":
			return "$(genDir)"
		case "bob_config":
			return configPath
		case "bob_config_opts":
			return configOpts
		default:
			return ""
		}
	})
	return
}

func (gc *generateCommon) createGenrule(mctx android.TopDownMutatorContext,
	out []string, depfile string) {

	// Map to Soong's genrule, where a single command produces the
	// outputs from the inputs.
	//
	// Setup a genrule property struct as if blueprint had read it
	// Only include the fields that we expect to use
	genProps := struct {
		Name         *string
		Srcs         []string
		Exclude_srcs []string
		Out          []string
		Depfile      *bool

		Tool_files []string
		Tools      []string
		Cmd        *string

		Export_include_dirs []string

		Owner       *string
		Proprietary *bool

		Enabled *bool
	}{}

	genProps.Name = proptools.StringPtr(gc.SimpleName.Name())
	genProps.Srcs = gc.Properties.Srcs
	genProps.Exclude_srcs = gc.Properties.Exclude_srcs
	genProps.Out = out
	genProps.Export_include_dirs = gc.Properties.Export_gen_include_dirs
	genProps.Enabled = gc.Properties.Enabled

	// Only set Tool_files or Tool if the Bob property is not ""
	// otherwise Soong will report a missing dependency
	if gc.Properties.Tool != "" {
		genProps.Tool_files = []string{gc.Properties.Tool}
	}
	if gc.Properties.Host_bin != "" {
		genProps.Tools = []string{gc.Properties.Host_bin}
	}

	// Bob's specified filename will be ignored. Soong will report an
	// error if $(depfile) is not used in the command
	genProps.Depfile = proptools.BoolPtr(depfile != "")

	// Replace ${args} immediately
	cmd := strings.Replace(gc.Properties.Cmd, "${args}",
		strings.Join(gc.Properties.Args, " "), -1)

	cmd2, err := expandBobVariables(cmd, gc.Properties.Tool, gc.Properties.Host_bin)
	if err != nil {
		panic(fmt.Errorf("%s property cmd %s", mctx.ModuleName(), err.Error()))
	}
	genProps.Cmd = proptools.StringPtr(cmd2)

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(android.ModuleFactoryAdaptor(genrule.GenRuleFactory), &genProps)
}

func (gs *generateSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	// Flatten features and expand templates
	featureApplierMutator(mctx, gs)
	templateApplierMutator(mctx, gs)

	gs.createGenrule(mctx, gs.Properties.Out, gs.Properties.Depfile)
}

func (gs *generateStaticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	// Flatten features and expand templates
	featureApplierMutator(mctx, gs)
	templateApplierMutator(mctx, gs)

	name := gs.SimpleName.Name()
	gs.createGenrule(mctx, []string{name + ".a"}, "")
}

func (gs *generateSharedLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	// Flatten features and expand templates
	featureApplierMutator(mctx, gs)
	templateApplierMutator(mctx, gs)

	name := gs.SimpleName.Name()
	gs.createGenrule(mctx, []string{name + ".so"}, "")
}

func (gb *generateBinary) soongBuildActions(mctx android.TopDownMutatorContext) {
	// Flatten features and expand templates
	featureApplierMutator(mctx, gb)
	templateApplierMutator(mctx, gb)

	name := gb.SimpleName.Name()
	gb.createGenrule(mctx, []string{name}, "")
}

var (
	// Use raw string literal backtick to avoid having to escape the
	// backslash in the regular expressions
	varRegexp = regexp.MustCompile(`\$[0-9]+`)
	dotRegexp = regexp.MustCompile(`\.{2,}`)
	extRegexp = regexp.MustCompile(`^\.`)
)

// Bob's module type allows the output file name to be specified using
// a regular expression replace, whereas Soong only allows you to
// specify the a new extension for the output.
//
// Look for an extension Bob's replacement string, and just use that. The
// replacement output filename won't be precisely as specified. If we
// want to maintain Bob behaviour we will need our own module type
// based on gensrcs.
func soongOutputExtension(re string) string {
	// We have a regular expression which might look like one of
	//  $1.ext
	//  $1.$2.ext
	//  $1.infix.$2.ext
	//  dir/$1.ext
	//
	// Drop the directory, match parts, and first '.', so we just end
	// up with `ext` or `infix.ext`. This should be good enough to keep
	// the files unique, and hopefully won't upset anything.
	// Drop directory
	dirChr := strings.LastIndex(re, "/")
	if dirChr > -1 {
		re = re[dirChr+1:]
	}
	// Remove capture group references, $[0-9]+
	re = varRegexp.ReplaceAllLiteralString(re, "")
	// With the capture group references removed, eliminate '.' which
	// are now adjacent by replacing .. with .
	// Note that this could cause breakage if we actually have '..'
	// (parent dir) in the replacement string, but that shouldn't be
	// happening.
	re = dotRegexp.ReplaceAllLiteralString(re, ".")
	// Trim initial '.'
	return extRegexp.ReplaceAllLiteralString(re, "")
}

func (ts *transformSource) soongBuildActions(mctx android.TopDownMutatorContext) {
	// Flatten features and expand templates.
	featureApplierMutator(mctx, ts)
	templateApplierMutator(mctx, ts)

	// bob_transform_source maps best to gensrcs
	//
	// Setup a gensrcs property struct as if blueprint had read it
	// Only include the fields that we expect to use
	transformProps := struct {
		Name             *string
		Srcs             []string
		Exclude_srcs     []string
		Output_extension *string
		Depfile          *bool

		Tool_files []string
		Tools      []string
		Cmd        *string

		Export_include_dirs []string

		Owner       *string
		Proprietary *bool

		Enabled *bool
	}{}

	transformProps.Name = proptools.StringPtr(ts.SimpleName.Name())
	transformProps.Srcs = ts.generateCommon.Properties.Srcs
	transformProps.Exclude_srcs = ts.generateCommon.Properties.Exclude_srcs
	transformProps.Export_include_dirs = ts.generateCommon.Properties.Export_gen_include_dirs
	transformProps.Enabled = ts.generateCommon.Properties.Enabled

	// Only set Tool_files or Tool if the Bob property is not ""
	// otherwise Soong will report a missing dependency
	if ts.generateCommon.Properties.Tool != "" {
		transformProps.Tool_files = []string{ts.generateCommon.Properties.Tool}
	}
	if ts.generateCommon.Properties.Host_bin != "" {
		transformProps.Tools = []string{ts.generateCommon.Properties.Host_bin}
	}

	// Bob's specified filename will be ignored. Soong will report an
	// error if $(depfile) is not used in the command
	transformProps.Depfile = proptools.BoolPtr(ts.Properties.Out.Depfile != "")

	if len(ts.Properties.Out.Replace) > 1 {
		panic(fmt.Errorf("Multiple outputs not supported in bob_transform_source on soong, %s", mctx.ModuleName()))
	}
	transformProps.Output_extension =
		proptools.StringPtr(soongOutputExtension(ts.Properties.Out.Replace[0]))

	// Replace ${args} immediately
	cmd := strings.Replace(ts.generateCommon.Properties.Cmd, "${args}",
		strings.Join(ts.generateCommon.Properties.Args, " "), -1)

	cmd2, err := expandBobVariables(cmd, ts.generateCommon.Properties.Tool,
		ts.generateCommon.Properties.Host_bin)
	if err != nil {
		panic(fmt.Errorf("%s property cmd %s", mctx.ModuleName(), err.Error()))
	}
	transformProps.Cmd = proptools.StringPtr(cmd2)

	// The ModuleDir for the new module will be inherited from the
	// current module via the TopDownMutatorContext
	mctx.CreateModule(android.ModuleFactoryAdaptor(genrule.GenSrcsFactory), &transformProps)
}
