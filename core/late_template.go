/*
 * Copyright 2019-2020 Arm Limited.
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
	"regexp"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"github.com/ARM-software/bob-build/internal/utils"
)

var matchSourcesRegex = regexp.MustCompile(`\{\{match_srcs\s+(.+?)\}\}`)

func (s *SourceProps) matchSources(ctx blueprint.BaseModuleContext, arg string,
	matchedNonCompiledSources map[string]bool) string {

	g := getBackend(ctx)

	for _, match := range matchSourcesRegex.FindAllStringSubmatch(arg, -1) {
		if len(match) <= 0 {
			panic("Invalid argument for match_srcs. match_srcs expects a single pattern used to match a file.")
		}
		matchedSources := []string{}
		for _, src := range s.getSources(ctx) {
			matched, err := pathtools.Match("**/"+match[1], src)
			if err != nil {
				panic("Error during matching filepath pattern")
			}
			if matched {
				matchedNonCompiledSources[src] = true
				matchedSources = append(matchedSources, getBackendPathInSourceDir(g, src))
			}
		}
		if len(matchedSources) == 0 {
			panic(fmt.Errorf("Could not match '%s' for module '%s'", match[1], ctx.ModuleName()))
		}
		arg = strings.Replace(arg, match[0], strings.Join(matchedSources, " "), 1)
	}
	return arg
}

// This mutator handles {{match_srcs}}. It returns the result of the
// input glob when applied to the modules source list. Because it
// needs access to the source list, this runs much later than other
// templates.
//
// This template is only applied in specific properties where we've
// seen sensible use-cases:
// - Build Props:
//  - Ldflags
// - Generated Common:
//  - Args
//  - Cmd
func matchSourcesMutator(mctx blueprint.TopDownMutatorContext) {
	module := mctx.Module()
	matchSrcsString := "{{match_srcs "
	if e, ok := module.(enableable); ok {
		if !isEnabled(e) {
			// Not enabled, skip execution
			return
		}
	}
	propArr := []*[]string{}
	propStr := []*string{}
	errorArrays := []*[]string{}
	var sourceProps *SourceProps
	if gsc, ok := getGenerateCommon(module); ok {
		propArr = []*[]string{&gsc.Properties.Args}
		propStr = []*string{gsc.Properties.Cmd}
		sourceProps = &gsc.Properties.SourceProps
	} else if buildProps, ok := module.(moduleWithBuildProps); ok {
		b := buildProps.build()
		propArr = []*[]string{&b.Ldflags}
		errorArrays = []*[]string{&b.Export_ldflags}
		propStr = []*string{}
		sourceProps = &b.SourceProps
	}

	// Unused non-compiled sources are not allowed, so create
	// a map to mark whether a non-compiled source is matched.
	matchedNonCompiledSources := make(map[string]bool)
	if _, ok := getLibrary(module); ok {
		for _, src := range sourceProps.getSources(mctx) {
			if utils.IsNotCompilableSource(src) {
				matchedNonCompiledSources[src] = false
			}
		}
	}

	for _, prop := range propArr {
		for i := range *prop {
			(*prop)[i] = sourceProps.matchSources(mctx, (*prop)[i], matchedNonCompiledSources)
		}
	}
	for _, prop := range propStr {
		*prop = sourceProps.matchSources(mctx, *prop, matchedNonCompiledSources)
	}
	for _, prop := range errorArrays {
		for i := range *prop {
			if strings.Contains((*prop)[i], matchSrcsString) {
				panic("Match_srcs not supported for exported variables.")
			}
		}
	}

	// Ensure that all non-compiled sources have been matched.
	for src, matched := range matchedNonCompiledSources {
		if !matched {
			panic(fmt.Errorf("Non-compiled source %s is not used by match_srcs.", src))
		}
	}
}
