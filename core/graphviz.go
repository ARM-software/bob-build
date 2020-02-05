/*
 * Copyright 2018, 2020 Arm Limited.
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
	"flag"
	"os"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/internal/graph"
	"github.com/ARM-software/bob-build/internal/utils"
)

var (
	graphOut             string
	graphWhoUses         string
	graphDependencies    string
	graphMark            string
	graphShowDefaults    bool
	graphShowBinaries    bool
	graphShowWholeStatic bool
	graphShowStaticLibs  bool
	graphShowSharedLibs  bool
)

func init() {
	flag.StringVar(&graphOut, "graph_out", "", "Output file name for dependency graph")
	flag.StringVar(&graphWhoUses, "graph_who_uses", "", "List of nodes to see in graph who uses them. If not set will be set as graph_out")
	flag.StringVar(&graphDependencies, "graph_dependencies", "", "List of nodes to see in graph what dependencies they have. If not set will be set as graph_out")
	flag.StringVar(&graphMark, "graph_mark", "", "List of nodes which should be marked. If empty this will contain who_uses + dependencies")
	flag.BoolVar(&graphShowDefaults, "graph_show_defaults", true, "Show default targets in graph")
	flag.BoolVar(&graphShowBinaries, "graph_show_binaries", true, "Show binaries targets in graph")
	flag.BoolVar(&graphShowWholeStatic, "graph_show_whole_static", true, "Show whole_static targets in graph")
	flag.BoolVar(&graphShowStaticLibs, "graph_show_static_libs", true, "Show static_libs targets in graph")
	flag.BoolVar(&graphShowSharedLibs, "graph_show_shared_libs", true, "Show shared_libs targets in graph")
}

type graphvizHandler struct {
	graph               graph.Graph
	whoUsesNodes        []string
	dependenciesOfNodes []string
	markNodes           []string
	showDefaults        bool
	showBinaries        bool
	showWholeStatic     bool
	showStaticLibraries bool
	showSharedLibraries bool
}

func initGrapvizHandler() *graphvizHandler {
	if len(graphOut) < 1 {
		return nil
	}

	if len(graphWhoUses) < 1 && len(graphDependencies) < 1 {
		graphWhoUses = graphOut
		graphDependencies = graphOut
	}
	if len(graphMark) < 1 {
		graphMark = graphWhoUses + "," + graphDependencies
	}

	return &graphvizHandler{graph.NewGraph(graphOut),
		utils.Trim(strings.Split(graphWhoUses, ",")),
		utils.Trim(strings.Split(graphDependencies, ",")),
		utils.Trim(strings.Split(graphMark, ",")),
		graphShowDefaults, graphShowBinaries, graphShowWholeStatic, graphShowStaticLibs, graphShowSharedLibs}
}

func (handler *graphvizHandler) generateGraphviz() {
	outputGraph := graph.NewGraph(handler.graph.GetName())
	for _, subgraph := range graph.GetSubgraphs(handler.graph) {
		for _, element := range handler.whoUsesNodes {
			if utils.Contains(subgraph.GetNodes(), element) {
				outputGraph.Merge(subgraph)
			}
		}
	}
	for _, element := range handler.dependenciesOfNodes {
		dependencySubgraph := graph.GetSubgraph(handler.graph, element)
		outputGraph.Merge(dependencySubgraph)
	}

	// Trim tree
	for _, element := range handler.whoUsesNodes {
		if !utils.Contains(handler.dependenciesOfNodes, element) {
			outputGraph.CutSubgraph(element)
		}
	}

	file, _ := os.Create(outputGraph.GetName() + ".graph")
	defer file.Close()
	file.WriteString(graph.ToString(outputGraph))
}

func (handler *graphvizHandler) graphvizMutator(mctx blueprint.BottomUpMutatorContext) {
	mainModule := mctx.Module()
	if e, ok := mainModule.(enableable); ok {
		if !isEnabled(e) {
			return // Not enabled, so not needed
		}
	}

	// Set type of node
	switch mainModule.(type) {
	case *staticLibrary:
		if !handler.showStaticLibraries {
			return
		}
		handler.graph.SetNodeBackgroundColor(mainModule.Name(), "green")
	case *sharedLibrary:
		if !handler.showSharedLibraries {
			return
		}
		handler.graph.SetNodeBackgroundColor(mainModule.Name(), "orange")
	case *binary:
		if !handler.showBinaries {
			return
		}
		handler.graph.SetNodeBackgroundColor(mainModule.Name(), "gray")
	case *defaults:
		if !handler.showDefaults {
			return
		}
		handler.graph.SetNodeBackgroundColor(mainModule.Name(), "yellow")
	}

	if utils.Contains(handler.markNodes, mainModule.Name()) {
		handler.graph.SetNodeProperty(mainModule.Name(), "shape", "doublecircle")
	}

	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		mainBuild := buildProps.build()

		if handler.showSharedLibraries {
			for _, lib := range mainBuild.Shared_libs {
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "orange")
			}

			for _, lib := range mainBuild.Export_shared_libs {
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "orange")
				handler.graph.SetEdgeProperty(mainModule.Name(), lib, "style", "dashed")
			}
		}

		if handler.showStaticLibraries {
			for _, lib := range mainBuild.Static_libs {
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "green")
			}

			for _, lib := range mainBuild.Export_static_libs {
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "green")
				handler.graph.SetEdgeProperty(mainModule.Name(), lib, "style", "dashed")
			}
		}

		for _, lib := range mainBuild.Whole_static_libs {
			handler.graph.AddEdge(mainModule.Name(), lib)
			handler.graph.SetEdgeColor(mainModule.Name(), lib, "red")
		}

		if !handler.showWholeStatic {
			for _, lib := range mainBuild.Whole_static_libs {
				handler.graph.DeleteProxyEdge(mainModule.Name(), lib)
			}
		}
	}

	if moduleDefault, ok := mainModule.(*defaults); ok && handler.showDefaults {
		for _, element := range moduleDefault.defaults() {
			handler.graph.AddEdge(mainModule.Name(), element)
			handler.graph.SetEdgeColor(mainModule.Name(), element, "yellow")
		}

	}
}

type quitSingleton struct {
	handler *graphvizHandler
}

func (m *quitSingleton) GenerateBuildActions(ctx blueprint.SingletonContext) {
	m.handler.generateGraphviz()
	os.Exit(0)
}

func (handler *graphvizHandler) quitSingletonFactory() blueprint.Singleton {
	return &quitSingleton{handler}
}
