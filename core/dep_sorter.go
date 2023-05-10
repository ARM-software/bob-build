/*
 * Copyright 2023 Arm Limited.
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
	"github.com/ARM-software/bob-build/internal/graph"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type graphMutatorHandler struct {
	graphs map[TgtType]graph.Graph
}

const (
	maxInt = int(^uint(0) >> 1)
	minInt = -maxInt - 1
)

func (handler *graphMutatorHandler) ResolveDependencySortMutator(ctx blueprint.BottomUpMutatorContext) {
	mainModule := ctx.Module()
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			return // Not enabled, so not needed
		}
	}
	if _, ok := mainModule.(*ModuleDefaults); ok {
		return // ignore bob_defaults
	}

	mainModuleName := mainModule.Name()

	if sp, ok := mainModule.(splittable); ok {
		if sp.getTarget() != "" {
			handler.graphs[sp.getTarget()].AddNode(mainModuleName)
		}
	}

	var mainBuild *Build
	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		mainBuild = buildProps.build()
	} else {
		return // ignore not a build
	}

	// This mutator is run after host/target splitting, so TargetType should have been set.
	if !(mainBuild.TargetType == tgtTypeTarget || mainBuild.TargetType == tgtTypeHost) {
		utils.Die("Cannot process dependencies on module '%s' with target type '%s'", mainModuleName, mainBuild.TargetType)
	}

	g := handler.graphs[mainBuild.TargetType]

	for _, lib := range mainBuild.Static_libs {
		if _, err := g.AddEdgeToExistingNodes(mainModuleName, lib); err != nil {
			utils.Die("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModuleName, lib, lib)
		}
		g.SetEdgeColor(mainModuleName, lib, "blue")
	}

	for _, lib := range mainBuild.Whole_static_libs {
		if _, err := g.AddEdgeToExistingNodes(mainModuleName, lib); err != nil {
			utils.Die("'%s' depends on '%s', but '%s' is either not defined or disabled", mainModuleName, lib, lib)
		}
		g.SetEdgeColor(mainModuleName, lib, "red")
	}

	temporaryPaths := map[string][]string{} // For preserving order in declaration

	for i, previous := range mainBuild.Static_libs {
		for j := i + 1; j < len(mainBuild.Static_libs); j++ {
			lib := mainBuild.Static_libs[j]
			if !g.IsReachable(lib, previous) {
				if g.AddEdge(previous, lib) {
					temporaryPaths[previous] = append(temporaryPaths[previous], lib)
					g.SetEdgeColor(previous, lib, "pink")
				}
			}
		}
	}

	sub := graph.GetSubgraph(g, mainModuleName)

	// Remove temporary path
	for key, list := range temporaryPaths {
		for _, value := range list {
			g.DeleteEdge(key, value)
		}
	}

	// The order of static libraries influences performance by
	// influencing memory layout. Where possible we want libraries
	// that depend on each other to be as close as possible. Library
	// order is determined by a topological sort.  Setting the
	// priority changes the order that child nodes are visited.
	//
	// Libraries that are frequently called are more
	// important and should be close to their callers. This information is not available in bob,
	// so estimate this with the number of users.
	//
	// Libraries that are large, or will cause a large number of
	// libraries to occur in the middle of the list, should be at the
	// end of the list. Treat this as the cost of visiting the
	// library. As an estimate of cost, count the number of libraries
	// that would be pulled in.
	//
	// The node priority is calculated as 'A * importance - cost',
	// where A is an arbitraty scaling factor.
	//
	// This is a bottom up mutator, so by the time we get to a binary
	// (or shared library), this mutator will have run on all their
	// dependencies and the (shared) graph will be complete (for the
	// current module).
	for _, nodeID := range sub.GetNodes() {
		cost := graph.GetSubgraphNodeCount(sub, nodeID)
		sources, _ := sub.GetSources(nodeID)
		priority := len(sources)
		sub.SetNodePriority(nodeID, (10*priority)-cost)
	}

	// The main library must always be evaluated first in the topological sort
	sub.SetNodePriority(mainModuleName, minInt)

	// We want those edges for calculating priority. After setting priority we can remove them.
	sub.DeleteProxyEdges("red")

	sub2 := graph.GetSubgraph(sub, mainModuleName)
	sortedStaticLibs, isDAG := graph.TopologicalSort(sub2)

	// Pop the module itself from the front of the list
	sortedStaticLibs = sortedStaticLibs[1:]

	if !isDAG {
		utils.Die("We have detected cycle: %s", mainModuleName)
	} else {
		mainBuild.ResolvedStaticLibs = sortedStaticLibs
	}

	extraStaticLibsDependencies := utils.Difference(mainBuild.ResolvedStaticLibs, mainBuild.Static_libs)

	ctx.AddVariationDependencies(nil, StaticTag, extraStaticLibsDependencies...)

	// This module may now depend on extra shared libraries, inherited from included
	// static libraries. Add that dependency here.
	ctx.AddVariationDependencies(nil, SharedTag, mainBuild.ExtraSharedLibs...)
}
