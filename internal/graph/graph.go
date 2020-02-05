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

package graph

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/ARM-software/bob-build/internal/utils"
)

type Attributes map[string]string

type node struct {
	priority       int
	id             string
	sources        map[string]bool
	edgeAttributes map[string]Attributes
	targets        []string
	attributes     Attributes
}

func NewNode(id string) *node {
	return &node{
		priority:       0,
		id:             id,
		sources:        make(map[string]bool),
		edgeAttributes: make(map[string]Attributes),
		targets:        []string{},
		attributes:     make(Attributes),
	}
}

func (n *node) ID() string {
	return n.id
}

func (n *node) hasSource(source string) bool {
	_, has := n.sources[source]
	return has
}

func (n *node) hasTarget(target string) bool {
	_, has := n.edgeAttributes[target]
	return has
}

type Graph interface {
	GetNodeCount() int

	GetName() string

	HasNode(id string) bool
	HasEdge(source, target string) bool

	GetNodes() []string

	SetNodeProperty(id, key, value string)
	SetNodeAttributes(id string, attributes Attributes)
	SetNodeBackgroundColor(id, color string)

	GetNodeAttributes(id string) (Attributes, error)

	Merge(mergeGraph Graph)
	copyNode(copyFrom Graph, node string)
	copyEdge(copyFrom Graph, source, target string)

	GetNodePriority(id string) (int, error)
	SetNodePriority(id string, priority int) error

	// return false if the node already present in the graph
	AddNode(id string) bool

	// return true if successfully removed from graph
	DeleteNode(id string) bool
	CutSubgraph(root string)

	// Removes node in this way that will keep connected parents to children of the node
	// eg. A -> B -> C, if we remove B then we will get A -> C
	DeleteProxyNode(id string)

	// Remove an edge, and replicate the outgoing connections of the target node onto the source node.
	// The edge being removed is a proxy for these connections. This is similar to DeleteProxyNode, but only deletes one of the incoming edges.
	DeleteProxyEdge(source, target string)
	// DeleteProxyEdge for edges that have a particular color
	DeleteProxyEdges(color string)
	// Like DeleteProxyEdge but additionally set edge color to replicate ones
	DeleteProxyEdgeSetColor(source, proxy, color string)

	GetEdgeAttributes(source, target string) (Attributes, error)
	SetEdgeAttributes(source, target string, attributes Attributes) bool

	AddEdge(source, target string) bool
	AddEdgeToExistingNodes(source, target string) (bool, error)
	SetEdgeColor(source, target string, color string)
	SetEdgeWeight(source, target string, weight int)
	SetEdgeConstraint(source, target string, constraint bool)
	SetEdgeProperty(source, target, key, value string)

	DeleteEdge(source, target string) error

	// GetSources returns the list of parent Nodes.
	// (Nodes that come towards the argument vertex.)
	GetSources(id string) ([]string, error)

	// GetTargets returns the list of child Nodes.
	// (Nodes that go out of the argument vertex.)
	GetTargets(id string) ([]string, error)

	IsReachable(source, target string) bool
}

type graph struct {
	mutex sync.RWMutex
	name  string

	// nodeMap stores all nodes.
	nodeMap map[string]*node
}

func (g *graph) CutSubgraph(root string) {
	targets, _ := g.GetTargets(root)
	for _, target := range targets {
		g.CutSubgraph(target)
		g.DeleteNode(target)
	}
}

func (g *graph) SetNodeAttributes(id string, attributes Attributes) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.addNode(id)
	g.nodeMap[id].attributes = attributes
}

func (g *graph) GetNodeAttributes(id string) (Attributes, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.hasNode(id) {
		return g.nodeMap[id].attributes, nil
	}
	return nil, fmt.Errorf("%s does not exist in the graph", id)
}

func (g *graph) SetNodeBackgroundColor(id, color string) {
	g.SetNodeProperty(id, "fillcolor", color)
	g.SetNodeProperty(id, "style", "filled")
}

func (g *graph) SetNodeProperty(id, key, value string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.addNode(id)
	g.nodeMap[id].attributes[key] = value
}

func (g *graph) Merge(mergeGraph Graph) {
	for _, node := range mergeGraph.GetNodes() {
		g.copyNode(mergeGraph, node)

		// We don't have to go through sources
		targets, _ := mergeGraph.GetTargets(node)
		for _, target := range targets {
			g.copyNode(mergeGraph, target)
			g.copyEdge(mergeGraph, node, target)
		}
	}
}

func (g *graph) copyNode(copyFrom Graph, node string) {
	g.AddNode(node)
	attributes, err := copyFrom.GetNodeAttributes(node)
	if err == nil {
		priority, _ := copyFrom.GetNodePriority(node)
		g.SetNodeAttributes(node, attributes)
		g.SetNodePriority(node, priority)
	}
}

func (g *graph) copyEdge(copyFrom Graph, source, target string) {
	g.AddEdge(source, target)
	attributes, _ := copyFrom.GetEdgeAttributes(source, target)
	g.SetEdgeAttributes(source, target, attributes)
}

func NewGraph(name string) Graph {
	return &graph{
		name:    name,
		nodeMap: make(map[string]*node),
	}
}

func (g *graph) GetName() string {
	return g.name
}

func (g *graph) GetNodeCount() int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return len(g.nodeMap)
}

func (g *graph) GetNodes() []string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	var copyOfNodes []string
	for key := range g.nodeMap {
		copyOfNodes = append(copyOfNodes, key)
	}

	return copyOfNodes
}

func (g *graph) hasNode(id string) bool {
	_, has := g.nodeMap[id]
	return has
}

func (g *graph) HasNode(id string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.hasNode(id)
}

func (g *graph) hasEdge(source, target string) bool {
	if !g.hasNode(source) {
		return false
	}
	if !g.hasNode(target) {
		return false
	}

	if _, ok := g.nodeMap[source].edgeAttributes[target]; ok {
		return true
	}
	return false
}

func (g *graph) HasEdge(source, target string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return g.hasEdge(source, target)
}

func (g *graph) GetNodePriority(id string) (int, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(id) {
		return 0, fmt.Errorf("%s does not exist in the graph", id)
	}

	return g.nodeMap[id].priority, nil
}

func (g *graph) SetNodePriority(id string, priority int) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(id) {
		return fmt.Errorf("%s does not exist in the graph", id)
	}

	g.nodeMap[id].priority = priority

	return nil
}

func (g *graph) AddNode(id string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.addNode(id)
}

func (g *graph) addNode(id string) bool {
	if g.hasNode(id) {
		return false
	}
	g.nodeMap[id] = NewNode(id)
	return true
}

func (g *graph) DeleteNode(id string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(id) {
		return false
	}

	// Remove all edges "from" this node (A) eg. A -> B, A -> C
	for child := range g.nodeMap[id].edgeAttributes {
		delete(g.nodeMap[child].sources, id)
	}

	// Remove all edges "to" this node (A) eg. B -> A, C -> A
	for parent := range g.nodeMap[id].sources {
		delete(g.nodeMap[parent].edgeAttributes, id)
		g.nodeMap[parent].targets = utils.Remove(g.nodeMap[parent].targets, id)
	}

	// Remove node itself
	delete(g.nodeMap, id)

	return true
}

func (g *graph) DeleteProxyNode(id string) {
	if !g.HasNode(id) {
		return
	}

	if sources, ok := g.GetSources(id); ok == nil {
		if targets, ok2 := g.GetTargets(id); ok2 == nil {
			for _, source := range sources {
				for _, target := range targets {
					g.AddEdge(source, target)
					attributes, _ := g.GetEdgeAttributes(id, target)
					g.SetEdgeAttributes(source, target, attributes)
				}
			}
		}
	}

	g.DeleteNode(id)
}

func (g *graph) deleteProxyEdgeSetColor(source, proxy, color string) {
	if targets, ok2 := g.getTargets(proxy); ok2 == nil {
		for _, target := range targets {
			g.addEdge(source, target)
			attributes, _ := g.getEdgeAttributes(proxy, target)
			g.setEdgeAttributes(source, target, attributes)
			if len(color) > 0 {
				g.setEdgeColor(source, target, color)
			}
		}
	}

	g.deleteEdge(source, proxy)
}

func (g *graph) DeleteProxyEdges(color string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	attrVal := "\"" + color + "\""

	// Re-evaluate the lists if we've removed edges to ensure we propogate multiple links
	removed := 1
	for removed > 0 {
		removed = 0
		for _, src := range g.nodeMap {
			targets, _ := g.getTargets(src.ID())
			for _, targetName := range targets {
				if src.edgeAttributes[targetName]["color"] == attrVal {
					g.deleteProxyEdgeSetColor(src.id, targetName, "")
					removed++
				}
			}
		}
	}
}

func (g *graph) DeleteProxyEdge(source, proxy string) {
	g.DeleteProxyEdgeSetColor(source, proxy, "")
}

func (g *graph) DeleteProxyEdgeSetColor(source, proxy, color string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(source) {
		return
	}
	if !g.hasNode(proxy) {
		return
	}

	if !g.hasEdge(source, proxy) {
		return // No such edge
	}

	g.deleteProxyEdgeSetColor(source, proxy, color)
}

func (g *graph) getEdge(source, target string) (*Attributes, error) {

	if !g.hasNode(source) {
		return nil, fmt.Errorf("%s does not exist in the graph", source)
	}
	if !g.hasNode(target) {
		return nil, fmt.Errorf("%s does not exist in the graph", target)
	}

	if attributes, ok := g.nodeMap[source].edgeAttributes[target]; ok {
		return &attributes, nil
	}
	return nil, fmt.Errorf("%s -> %s does not exist in the graph", source, target)
}

func (g *graph) IsReachable(source, target string) bool {
	if !g.HasNode(source) {
		return false
	}
	if !g.HasNode(target) {
		return false
	}

	sub := GetSubgraph(g, source)
	return sub.HasNode(target)
}

func (g *graph) getEdgeAttributes(source, target string) (Attributes, error) {
	attributes, err := g.getEdge(source, target)
	if err == nil {
		return *attributes, nil
	}
	return nil, err
}

func (g *graph) GetEdgeAttributes(source, target string) (Attributes, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return g.getEdgeAttributes(source, target)
}

func (g *graph) setEdgeAttributes(source, target string, attributes Attributes) bool {
	if _, ok := g.nodeMap[source].edgeAttributes[target]; ok {
		g.nodeMap[source].edgeAttributes[target] = attributes
		return true
	}
	return false
}

func (g *graph) SetEdgeAttributes(source, target string, attributes Attributes) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(source) {
		return false
	}
	if !g.hasNode(target) {
		return false
	}

	return g.setEdgeAttributes(source, target, attributes)
}

func (g *graph) addEdge(source, target string) bool {
	if g.nodeMap[source].hasTarget(target) {
		return false
	}

	g.nodeMap[source].targets = append(g.nodeMap[source].targets, target)
	g.nodeMap[source].edgeAttributes[target] = Attributes{}
	g.nodeMap[target].sources[source] = true
	return true
}

func (g *graph) AddEdge(source, target string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(source) {
		g.addNode(source)
	}
	if !g.hasNode(target) {
		g.addNode(target)
	}

	return g.addEdge(source, target)
}

func (g *graph) setEdgeColor(source, target string, color string) {
	g.setEdgeProperty(source, target, "color", "\""+color+"\"")
}

func (g *graph) AddEdgeToExistingNodes(source, target string) (bool, error) {
	if !g.HasNode(source) {
		return false, fmt.Errorf("'%s' does not exist in the graph", target)
	}
	if !g.HasNode(target) {
		return false, fmt.Errorf("'%s' does not exist in the graph", target)
	}

	return g.AddEdge(source, target), nil
}

func (g *graph) SetEdgeColor(source, target string, color string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.setEdgeColor(source, target, color)
}

func (g *graph) SetEdgeWeight(source, target string, weight int) {
	g.SetEdgeProperty(source, target, "weight", strconv.Itoa(weight))
}

func (g *graph) SetEdgeConstraint(source, target string, constraint bool) {
	if constraint {
		g.SetEdgeProperty(source, target, "constraint", "\"true\"")
	} else {
		g.SetEdgeProperty(source, target, "constraint", "\"false\"")
	}
}

func (g *graph) setEdgeProperty(source, target, key, value string) {
	if attributes, ok := g.getEdge(source, target); ok == nil {
		(*attributes)[key] = value
	}
}

func (g *graph) SetEdgeProperty(source, target, key, value string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.setEdgeProperty(source, target, key, value)
}

func (g *graph) deleteEdge(source, target string) {
	g.nodeMap[source].targets = utils.Remove(g.nodeMap[source].targets, target)
	delete(g.nodeMap[source].edgeAttributes, target)
	delete(g.nodeMap[target].sources, source)
}

func (g *graph) DeleteEdge(source, target string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.hasNode(source) {
		return fmt.Errorf("%s does not exist in the graph", source)
	}
	if !g.hasNode(target) {
		return fmt.Errorf("%s does not exist in the graph", target)
	}

	g.deleteEdge(source, target)

	return nil
}

func (g *graph) GetSources(id string) ([]string, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.getSources(id)
}

func (g *graph) getSources(id string) ([]string, error) {
	if !g.hasNode(id) {
		return nil, fmt.Errorf("%s does not exist in the graph", id)
	}

	copySources := []string{}
	for source := range g.nodeMap[id].sources {
		copySources = append(copySources, source)
	}
	return copySources, nil
}

func (g *graph) GetTargets(id string) ([]string, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.getTargets(id)
}

func (g *graph) getTargets(id string) ([]string, error) {
	if !g.hasNode(id) {
		return nil, fmt.Errorf("%s does not exist in the graph", id)
	}

	copyTargets := make([]string, len(g.nodeMap[id].targets))
	copy(copyTargets, g.nodeMap[id].targets)

	return copyTargets, nil
}

// Return graphviz string representation, we can output this to file and preview in eg. xdot or any other tool for graphviz
func ToString(graph Graph) string {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "digraph {\n")
	fmt.Fprintf(buf, "ranksep=2;\n")

	for _, id := range graph.GetNodes() {
		fmt.Fprintf(buf, "\t\"%s\" [", id)
		if attributes, ok := graph.GetNodeAttributes(id); ok == nil {
			for attrName, attrValue := range attributes {
				fmt.Fprintf(buf, "%s=%s,", attrName, attrValue)
			}
		}
		fmt.Fprintf(buf, "];\n")

		targets, _ := graph.GetTargets(id)
		for _, targetID := range targets {
			fmt.Fprintf(buf, "\t\"%s\" -> \"%s\" [", id, targetID)
			if attributes, ok := graph.GetEdgeAttributes(id, targetID); ok == nil {
				for attrName, attrValue := range attributes {
					fmt.Fprintf(buf, "%s=%s,", attrName, attrValue)
				}
			}
			fmt.Fprintf(buf, "];\n")
		}
	}

	fmt.Fprintf(buf, "}\n")
	return buf.String()
}

// Walk down in recursive way the graph like in DFS but beside walk we
// save this walk (simply copy walk path) to our new graph
func walkDown(graph Graph, walk Graph, nodeID string, visited map[string]bool) {
	visited[nodeID] = true

	walk.copyNode(graph, nodeID)
	targets, _ := graph.GetTargets(nodeID)

	for _, targetID := range targets {
		if walk.AddEdge(nodeID, targetID) {
			walk.copyNode(graph, targetID)
			walk.copyEdge(graph, nodeID, targetID)
			walkDown(graph, walk, targetID, visited)
		}
	}
}

// Retrive all possible SubGraphs. Check GetSubGraph to understand what is sub graph
func GetSubgraphs(graph Graph) []Graph {
	subGraphs := []Graph{}
	visited := make(map[string]bool)

	for _, id := range graph.GetNodes() {
		if sources, _ := graph.GetSources(id); len(sources) > 0 {
			continue
		}

		sub := NewGraph(id)
		walkDown(graph, sub, id, visited)
		subGraphs = append(subGraphs, sub)
	}

	// Go through cycled ones
	for _, id := range graph.GetNodes() {
		if visited[id] {
			continue
		}
		sub := NewGraph(id)
		walkDown(graph, sub, id, visited)
		subGraphs = append(subGraphs, sub)
	}

	return subGraphs
}

// A subgraph of a graph G is another graph formed from a subset of the vertices and edges of G.
// The vertex subset must include all endpoints of the edge subset.
// subgraph: https://en.wikipedia.org/wiki/Glossary_of_graph_theory_terms#subgraph
func GetSubgraph(graph Graph, start string) Graph {
	visited := make(map[string]bool)

	sub := NewGraph(start)
	walkDown(graph, sub, start, visited)

	return sub
}

type ByNodePriority struct {
	nodes []string
	g     Graph
}

func (a ByNodePriority) Len() int      { return len(a.nodes) }
func (a ByNodePriority) Swap(i, j int) { a.nodes[i], a.nodes[j] = a.nodes[j], a.nodes[i] }
func (a ByNodePriority) Less(i, j int) bool {
	pa, _ := a.g.GetNodePriority(a.nodes[i])
	pb, _ := a.g.GetNodePriority(a.nodes[j])
	return pa < pb
}

// Wiki: https://en.wikipedia.org/wiki/Topological_sorting
func TopologicalSort(g Graph) ([]string, bool) {
	L := []string{}
	isDAG := true // Directed Acyclic Graph (DAG)
	color := make(map[string]string)
	for _, v := range g.GetNodes() {
		color[v] = "white"
	}
	allNodes := g.GetNodes()
	sort.Stable(ByNodePriority{allNodes, g})

	// for each vertex v in G:
	for _, v := range allNodes {
		// if v.color == "white":
		if color[v] == "white" {
			// topologicalSortVisit(v, L, isDAG)
			topologicalSortVisit(g, v, &L, &isDAG, &color)
		}
	}

	return L, isDAG
}

func topologicalSortVisit(
	g Graph,
	id string,
	L *[]string,
	isDAG *bool,
	color *map[string]string,
) {
	// if v.color == "gray":
	if (*color)[id] == "gray" {
		// isDAG = false
		*isDAG = false
		return
	}

	// if v.color == "white":
	if (*color)[id] == "white" {
		// v.color = "gray":
		(*color)[id] = "gray"

		// for each child vertex w of v:
		cmap, err := g.GetTargets(id)
		if err != nil {
			panic(err)
		}
		cmap = utils.Reversed(cmap)
		sort.Stable(ByNodePriority{cmap, g})
		for _, w := range cmap {
			// topologicalSortVisit(w, L, isDAG)
			topologicalSortVisit(g, w, L, isDAG, color)
		}

		// v.color = "black"
		(*color)[id] = "black"

		// L.push_front(v)
		temp := make([]string, len(*L)+1)
		temp[0] = id
		copy(temp[1:], *L)
		*L = temp
	}
}
