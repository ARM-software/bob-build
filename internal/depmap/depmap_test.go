package depmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Basic(t *testing.T) {
	m := NewDepmap()
	deps := []string{"a", "b", "c"}
	key := "foo"
	m.SetDeps(key, deps)
	assert.Equal(t, deps, m.GetDeps(key))
}

func Test_Empty(t *testing.T) {
	m := NewDepmap()
	assert.Equal(t, []string{}, m.GetDeps("does_not_exist"))
}

func Test_Add(t *testing.T) {
	m := NewDepmap()
	deps1 := []string{"a", "b", "c"}
	deps2 := []string{"d", "e", "f"}
	key := "foo"
	m.SetDeps(key, deps1)
	m.AddDeps(key, deps2)
	assert.Equal(t, append(deps1, deps2...), m.GetDeps(key))
}

func Test_AddToEmpty(t *testing.T) {
	m := NewDepmap()
	deps := []string{"a", "b", "c"}
	key := "foo"
	m.AddDeps(key, deps)
	assert.Equal(t, deps, m.GetDeps(key))
}

func Test_TransativeSimple(t *testing.T) {
	m := NewDepmap()

	// Simple tree
	//       a
	//     /   \
	//    b     c
	//   / \   / \
	//   d e   g f

	m.SetDeps("a", []string{"b", "c"})
	m.SetDeps("b", []string{"d", "e"})
	m.SetDeps("c", []string{"g", "f"})

	// DFS order expected:
	assert.Equal(t, []string{"b", "d", "e", "c", "g", "f"}, m.GetAllDeps("a"))
	assert.Equal(t, []string{"d", "e"}, m.GetAllDeps("b"))
	assert.Equal(t, []string{"g", "f"}, m.GetAllDeps("c"))
}

func Test_Diamond(t *testing.T) {
	m := NewDepmap()

	//      a
	//     / \
	//    b - c
	//     \ /
	//      d

	m.SetDeps("a", []string{"b", "c"})
	m.SetDeps("b", []string{"c", "d"})
	m.SetDeps("c", []string{"b", "d"})

	assert.Equal(t, []string{"b", "c", "d"}, m.GetAllDeps("a"))
}

func Test_HandleCircularGracefully(t *testing.T) {
	m := NewDepmap()

	//      a
	//     / \
	//    b   c
	//     \ /
	//      a

	m.SetDeps("a", []string{"b", "c"})
	m.SetDeps("b", []string{"a"})
	m.SetDeps("c", []string{"a"})

	assert.Equal(t, []string{"b", "c"}, m.GetAllDeps("a"))
}

func Test_TransativeWithManyDuplicates(t *testing.T) {
	m := NewDepmap()

	m.SetDeps("a", []string{"b"})
	m.SetDeps("b", []string{"a", "c"})
	m.SetDeps("c", []string{"a", "d"})
	m.SetDeps("d", []string{"a", "e"})
	m.SetDeps("e", []string{"a"})

	assert.Equal(t, []string{"b", "c", "d", "e"}, m.GetAllDeps("a"))
}

func Test_Traverse(t *testing.T) {
	m := NewDepmap()

	m.SetDeps("a", []string{"b"})
	m.SetDeps("b", []string{"a", "c"})
	m.SetDeps("c", []string{"a", "d"})
	m.SetDeps("d", []string{"a", "e"})
	m.SetDeps("e", []string{"a"})

	visited := map[string]int{}
	loops := map[string]int{}

	expect_visited := map[string]int{"b": 1, "c": 1, "d": 1, "e": 1}
	expect_loops := map[string]int{"a": 4}

	assert.Equal(t, []string{"b", "c", "d", "e"}, m.GetAllDeps("a"))
	m.Traverse("a",
		func(k string) {
			visited[k] += 1
		},
		func(k string) {
			loops[k] += 1
		},
	)
	assert.Equal(t, expect_visited, visited)
	assert.Equal(t, expect_loops, loops)

}

func Test_TraverseEmpty(t *testing.T) {
	m := NewDepmap()

	m.Traverse("a",
		func(k string) {
			t.FailNow()
		},
		func(k string) {
			t.FailNow()
		},
	)
}
