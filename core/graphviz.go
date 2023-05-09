/*
 * Copyright 2018, 2020, 2023 Arm Limited.
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
	graphStartNodes      string
	graphOut             string
	graphShowReverseDeps bool
	graphShowDeps        bool
	graphShowDefaults    bool
	graphShowBinaries    bool
	graphShowWholeStatic bool
	graphShowStaticLibs  bool
	graphShowSharedLibs  bool
	graphShowLdlibs      bool
)

func init() {
	flag.StringVar(&graphStartNodes, "graph-start-nodes", "",
		"Comma separated list of initial nodes")
	flag.StringVar(&graphOut, "graph-out", "",
		"Output file name for dependency graph. Defaults to first graph-start-nodes")
	flag.BoolVar(&graphShowReverseDeps, "graph-rev-deps", false,
		"Show reverse dependencies (users) of graph-start-nodes")
	flag.BoolVar(&graphShowDeps, "graph-deps", true, "Show dependencies of graph-start-nodes")
	flag.BoolVar(&graphShowDefaults, "graph-show-defaults", false, "Show defaults modules in graph")
	flag.BoolVar(&graphShowBinaries, "graph-show-binaries", true, "Show binary modules in graph")
	flag.BoolVar(&graphShowWholeStatic, "graph-show-whole-static", true,
		"Show static libraries linked as whole_static")
	flag.BoolVar(&graphShowStaticLibs, "graph-show-static-libs", true, "Show static libraries")
	flag.BoolVar(&graphShowSharedLibs, "graph-show-shared-libs", true, "Show shared libraries")
	flag.BoolVar(&graphShowLdlibs, "graph-show-ldlibs", false, "Show ldlib usage")
}

type graphvizHandler struct {
	graph               graph.Graph
	startNodes          []string
	showReverseDeps     bool
	showDeps            bool
	showDefaults        bool
	showBinaries        bool
	showWholeStatic     bool
	showStaticLibraries bool
	showSharedLibraries bool
	showLdlibs          bool
}

func initGrapvizHandler() *graphvizHandler {
	if len(graphStartNodes) < 1 {
		return nil
	}

	if graphOut == "" {
		graphOut = strings.SplitN(graphStartNodes, ",", 2)[0] + ".graph"
	}

	return &graphvizHandler{graph.NewGraph(graphOut),
		utils.Trim(strings.Split(graphStartNodes, ",")),
		graphShowReverseDeps,
		graphShowDeps,
		graphShowDefaults,
		graphShowBinaries, graphShowWholeStatic, graphShowStaticLibs, graphShowSharedLibs,
		graphShowLdlibs}
}

func (handler *graphvizHandler) generateGraphviz() {
	outputGraph := graph.NewGraph(handler.graph.GetName())
	if handler.showReverseDeps {
		for _, subgraph := range graph.GetSubgraphs(handler.graph) {
			for _, element := range handler.startNodes {
				if utils.Contains(subgraph.GetNodes(), element) {
					outputGraph.Merge(subgraph)
				}
			}
		}
	}
	if handler.showDeps {
		for _, element := range handler.startNodes {
			dependencySubgraph := graph.GetSubgraph(handler.graph, element)
			outputGraph.Merge(dependencySubgraph)
		}
	}

	file, _ := os.Create(outputGraph.GetName())
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

	showLdlibs := handler.showLdlibs
	depEdgeStyle := "solid"

	// Set type of node
	switch mainModule.(type) {
	case *ModuleStaticLibrary:
		if !handler.showStaticLibraries {
			return
		}
		handler.graph.SetNodeBackgroundColor(mainModule.Name(), "green")

		// Don't show ldlibs usage on static libraries, as these
		// aren't actually applied
		showLdlibs = false
		depEdgeStyle = "dashed"
	case *ModuleSharedLibrary:
		if !handler.showSharedLibraries {
			return
		}
		handler.graph.SetNodeBackgroundColor(mainModule.Name(), "orange")
	case *ModuleBinary:
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

	if utils.Contains(handler.startNodes, mainModule.Name()) {
		handler.graph.SetNodeProperty(mainModule.Name(), "shape", "doublecircle")
	}

	if buildProps, ok := mainModule.(moduleWithBuildProps); ok {
		mainBuild := buildProps.build()

		if handler.showSharedLibraries {
			for _, lib := range mainBuild.Shared_libs {
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "orange")
				handler.graph.SetEdgeProperty(mainModule.Name(), lib, "style", depEdgeStyle)
			}
		}

		if handler.showStaticLibraries {
			for _, lib := range mainBuild.Static_libs {
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "green")
				handler.graph.SetEdgeProperty(mainModule.Name(), lib, "style", depEdgeStyle)
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

		if showLdlibs {
			for _, lib := range mainBuild.Ldlibs {
				handler.graph.SetNodeBackgroundColor(lib, "skyblue")
				handler.graph.AddEdge(mainModule.Name(), lib)
				handler.graph.SetEdgeColor(mainModule.Name(), lib, "skyblue")
				handler.graph.SetEdgeProperty(mainModule.Name(), lib, "style", depEdgeStyle)
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
