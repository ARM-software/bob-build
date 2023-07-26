package graph

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ARM-software/bob-build/internal/utils"
)

func TestShould_add_new_edge_When_edge_removed(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("3", "10")

	if testGraph.AddEdge("8", "9") {
		testGraph.SetEdgeColor("8", "9", "red")
	} else {
		t.Errorf("Edge should be new")
	}

	if testGraph.AddEdge("8", "9") {
		t.Errorf("Edge already added!")
	}

	testGraph.DeleteEdge("8", "9")

	if testGraph.AddEdge("8", "9") {
		testGraph.SetEdgeColor("8", "9", "blue")
	} else {
		t.Errorf("Edge should be new")
	}
}

func TestFun(t *testing.T) {
	////////////////////////////////////////////
	// The graph shown below has many valid topological sorts, including:

	//    5, 7, 3, 11, 8, 2, 9, 10 (visual left-to-right, top-to-bottom)
	//    3, 5, 7, 8, 11, 2, 9, 10 (smallest-numbered available vertex first)
	//    5, 7, 3, 8, 11, 10, 9, 2 (fewest edges first)
	//    7, 5, 11, 3, 10, 8, 9, 2 (largest-numbered available vertex first) <---------
	//    5, 7, 11, 2, 3, 8, 9, 10 (attempting top-to-bottom, left-to-right)
	//    3, 7, 8, 5, 11, 10, 2, 9 (arbitrary)
	testGraph := NewGraph("Test")
	testGraph.AddEdge("3", "10")
	testGraph.SetEdgeColor("3", "10", "blue")
	testGraph.AddEdge("11", "2")
	testGraph.AddEdge("11", "9")
	testGraph.AddEdge("11", "10")
	testGraph.AddEdge("7", "8")
	testGraph.AddEdge("3", "8")
	testGraph.AddEdge("5", "11")
	testGraph.AddEdge("7", "11")
	testGraph.AddEdge("8", "9")

	for i, sub := range GetSubgraphs(testGraph) {
		t.Log("Index: " + strconv.Itoa(i) + " - " + sub.GetName())
		t.Log(ToString(sub))
	}

	t.Log(testGraph.GetName())
	t.Log(ToString(testGraph))

	sorted, _ := TopologicalSort(testGraph)

	// Check that each condition is met
	if utils.Find(sorted, "3") > utils.Find(sorted, "10") {
		t.Errorf("3 and 10 not correctly sorted")
	}

	if utils.Find(sorted, "11") > utils.Find(sorted, "2") {
		t.Errorf("11 and 2 not correctly sorted")
	}

	if utils.Find(sorted, "11") > utils.Find(sorted, "9") {
		t.Errorf("11 and 9 not correctly sorted")
	}

	if utils.Find(sorted, "11") > utils.Find(sorted, "10") {
		t.Errorf("11 and 10 not correctly sorted")
	}

	if utils.Find(sorted, "7") > utils.Find(sorted, "8") {
		t.Errorf("7 and 8 not correctly sorted")
	}

	if utils.Find(sorted, "3") > utils.Find(sorted, "8") {
		t.Errorf("13 and 8 not correctly sorted")
	}

	if utils.Find(sorted, "5") > utils.Find(sorted, "11") {
		t.Errorf("5 and 11 not correctly sorted")
	}

	if utils.Find(sorted, "7") > utils.Find(sorted, "11") {
		t.Errorf("7 and 11 not correctly sorted")
	}

	if utils.Find(sorted, "8") > utils.Find(sorted, "9") {
		t.Errorf("8 and 9 not correctly sorted")
	}
}

func TestShould_pass_connections_When_remove_proxy_node(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("A", "B")
	testGraph.AddEdge("B", "C")
	testGraph.SetEdgeColor("B", "C", "blue")

	testGraph.DeleteProxyNode("B")

	if testGraph.HasNode("B") {
		t.Errorf("Node shouldn't be there")
	}

	if testGraph.AddEdge("A", "C") {
		t.Errorf("Edge should already exist")
	}

	if attributes, err := testGraph.GetEdgeAttributes("A", "C"); err == nil {
		if attributes["color"] != "\"blue\"" {
			t.Logf("Edge color: %s", attributes["color"])
			t.Errorf("Edge A -> C should be blue")
		}
	} else {
		t.Errorf("Edge should already exist")
	}

}

func TestShould_pass_connections_When_remove_proxy_node2(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("A", "B")
	testGraph.AddEdge("X", "B")
	testGraph.AddEdge("B", "C")
	testGraph.AddEdge("B", "D")
	testGraph.AddEdge("B", "E")
	testGraph.SetEdgeColor("B", "C", "blue")

	testGraph.DeleteProxyNode("B")

	if testGraph.HasNode("B") {
		t.Errorf("Node shouldn't be there")
	}

	if testGraph.AddEdge("A", "C") {
		t.Errorf("Edge should already exist")
	}
	if testGraph.AddEdge("A", "D") {
		t.Errorf("Edge should already exist")
	}
	if testGraph.AddEdge("A", "E") {
		t.Errorf("Edge should already exist")
	}
	if testGraph.AddEdge("X", "C") {
		t.Errorf("Edge should already exist")
	}
	if attributes, err := testGraph.GetEdgeAttributes("X", "C"); err == nil {
		if attributes["color"] != "\"blue\"" {
			t.Logf("Edge color: %s", attributes["color"])
			t.Errorf("Edge X -> C should be blue")
		}
	} else {
		t.Errorf("Edge should already exist")
	}

	if testGraph.AddEdge("X", "D") {
		t.Errorf("Edge should already exist")
	}
	if testGraph.AddEdge("X", "E") {
		t.Errorf("Edge should already exist")
	}
}

func TestShould_pass_connections_When_remove_proxy_edge(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("A", "B")
	testGraph.AddEdge("X", "B")
	testGraph.AddEdge("B", "C")
	testGraph.AddEdge("B", "D")
	testGraph.AddEdge("B", "E")
	testGraph.SetEdgeColor("B", "C", "blue")

	testGraph.DeleteProxyEdge("X", "B")

	if !testGraph.HasNode("B") {
		t.Errorf("Node should be there")
	}

	if testGraph.HasEdge("A", "C") {
		t.Errorf("Edge shouldn't exist")
	}
	if testGraph.HasEdge("A", "D") {
		t.Errorf("Edge shouldn't exist")
	}
	if testGraph.HasEdge("A", "E") {
		t.Errorf("Edge shouldn't exist")
	}

	if !testGraph.HasEdge("X", "C") {
		t.Errorf("Edge should exist")
	}
	if attributes, err := testGraph.GetEdgeAttributes("X", "C"); err == nil {
		if attributes["color"] != "\"blue\"" {
			t.Logf("Edge color: %s", attributes["color"])
			t.Errorf("Edge X -> C should be blue")
		}
	} else {
		t.Errorf("Edge should already exist")
	}

	if !testGraph.HasEdge("X", "D") {
		t.Errorf("Edge should exist")
	}
	if !testGraph.HasEdge("X", "E") {
		t.Errorf("Edge should exist")
	}
}

func TestShould_not_pass_connections_When_remove_proxy_edge_isnt_connected(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("A", "B")
	testGraph.AddEdge("B", "C")
	testGraph.AddEdge("C", "X")
	testGraph.AddEdge("B", "D")
	testGraph.AddEdge("B", "E")

	if testGraph.HasEdge("A", "X") {
		t.Errorf("Edge should exist")
	}

	testGraph.DeleteProxyEdge("A", "C")

	if testGraph.HasEdge("A", "X") {
		t.Log(ToString(testGraph))
		t.Errorf("Edge should exist")
	}
}

func Test_TopologicalSortMaintainsSubnodeOrder(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("top", "a")
	testGraph.AddEdge("top", "b")
	testGraph.AddEdge("top", "c")
	testGraph.AddEdge("top", "d")
	testGraph.AddEdge("top", "e")
	testGraph.AddEdge("top", "f")

	// To ensure orderring, visit the top node first
	testGraph.SetNodePriority("top", -1)

	target := []string{"top", "a", "b", "c", "d", "e", "f"}

	sorted, _ := TopologicalSort(testGraph)
	assert.Equal(t, target, sorted)

	// If we copy the graph we should still get the same sort result
	subGraph := GetSubgraph(testGraph, "top")

	sorted, _ = TopologicalSort(subGraph)
	assert.Equal(t, target, sorted)
}

func Test_SubnodeOrderAfterDeleteProxyEdges(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("top", "a")
	testGraph.AddEdge("top", "b")
	testGraph.AddEdge("top", "c")
	testGraph.AddEdge("top", "d")

	testGraph.AddEdge("d", "df")
	testGraph.AddEdge("d", "de")
	testGraph.AddEdge("c", "cf")
	testGraph.AddEdge("c", "ce")
	testGraph.AddEdge("b", "bf")
	testGraph.AddEdge("b", "be")
	testGraph.AddEdge("a", "af")
	testGraph.AddEdge("a", "ae")

	testGraph.SetEdgeColor("top", "a", "red")
	testGraph.SetEdgeColor("top", "b", "red")
	testGraph.SetEdgeColor("top", "c", "red")
	testGraph.SetEdgeColor("top", "d", "red")

	testGraph.DeleteProxyEdges("red")

	// Remove unref'd nodes
	testGraph = GetSubgraph(testGraph, "top")

	// To ensure orderring, visit the top node first
	testGraph.SetNodePriority("top", -1)

	// ProxyEdges should be removed in subnode order, and their children added in order
	target := []string{"top", "af", "ae", "bf", "be", "cf", "ce", "df", "de"}

	sorted, _ := TopologicalSort(testGraph)
	assert.Equal(t, target, sorted)
}

func Test_GetSubgraphNodeCount(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("top", "a")
	testGraph.AddEdge("top", "b")

	testGraph.AddEdge("a", "a0")
	testGraph.AddEdge("a", "a1")
	testGraph.AddEdge("b", "b0")
	testGraph.AddEdge("b", "b1")

	nodes := []string{"top", "a", "a0", "a1", "b", "b0", "b1"}

	for _, node := range nodes {
		expected := GetSubgraph(testGraph, node).GetNodeCount()
		result := GetSubgraphNodeCount(testGraph, node)
		assert.Equal(t, expected, result)
	}
}

func Test_GetSubgraphHasNode(t *testing.T) {
	testGraph := NewGraph("Test")
	testGraph.AddEdge("top", "a")
	testGraph.AddEdge("top", "b")

	testGraph.AddEdge("a", "a0")
	testGraph.AddEdge("a", "a1")
	testGraph.AddEdge("b", "b0")
	testGraph.AddEdge("b", "b1")

	nodes := []string{"top", "a", "a0", "a1", "b", "b0", "b1"}

	for _, node1 := range nodes {
		for _, node2 := range nodes {
			expected := GetSubgraph(testGraph, node1).HasNode(node2)
			result := GetSubgraphHasNode(testGraph, node1, node2)
			assert.Equal(t, expected, result)
		}
	}
}
