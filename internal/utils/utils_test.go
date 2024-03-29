package utils

import (
	"fmt"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func Test_IsHeader(t *testing.T) {
	assert.True(t, IsHeader("bla.hpp"), "bla.hpp")
	assert.False(t, IsHeader("bla.bin"), "bla.bin")
}

func Test_IsNotHeader(t *testing.T) {
	assert.True(t, IsNotHeader("bla.c"), "bla.c")
	assert.False(t, IsNotHeader("bla.h"), "bla.h")
}

func Test_IsCompilableSource(t *testing.T) {
	assert.True(t, IsCompilableSource("bla.c"), "bla.c")
	assert.False(t, IsCompilableSource("bla.bbq"), "bla.bbq")
}

func Test_IsNotCompilableSource(t *testing.T) {
	assert.True(t, IsNotCompilableSource("bla.txt"), "bla.txt")
	assert.False(t, IsNotCompilableSource("bla.S"), "bla.S")
}

func Test_PrefixAll(t *testing.T) {
	if len(PrefixAll([]string{}, "myprefix")) != 0 {
		t.Errorf("Incorrect handling of empty list")
	}

	in := []string{"abc def", ";1234	;''"}
	prefix := "!>@@\""
	correct := []string{"!>@@\"abc def", "!>@@\";1234	;''"}

	assert.Equal(t, correct, PrefixAll(in, prefix))
}

func Test_StripPrefixAll(t *testing.T) {
	in := []string{":a", ":b", ":some:target"}
	prefix := ":"
	correct := []string{"a", "b", "some:target"}
	assert.Equal(t, correct, StripPrefixAll(in, prefix))
}

func Test_PrefixDirs(t *testing.T) {
	if len(PrefixDirs([]string{}, "myprefix")) != 0 {
		t.Errorf("Incorrect handling of empty list")
	}

	in := []string{"src/foo.c", "include/bar.h"}
	prefix := "$(LOCAL_PATH)"
	correct := []string{"$(LOCAL_PATH)/src/foo.c", "$(LOCAL_PATH)/include/bar.h"}

	assert.Equal(t, correct, PrefixDirs(in, prefix))
}

func Test_SortedKeys(t *testing.T) {
	in := map[string]string{
		"Zebra":    "grass",
		"aardvark": "insects",
		"./a.out":  "bits",
	}
	assert.Equal(t, []string{"./a.out", "Zebra", "aardvark"}, SortedKeys(in))
}

func Test_SortedKeysBoolMap(t *testing.T) {
	in := map[string]bool{
		"Alphabetic characters should appear after numbers": true,
		"2 + 2 = 5": false,
	}
	correct := []string{"2 + 2 = 5", "Alphabetic characters should appear after numbers"}
	out := SortedKeysBoolMap(in)

	assert.Equal(t, correct, out)
}

func Test_Contains(t *testing.T) {
	assert.Falsef(t, Contains([]string{"a", "b", "c"}, "yellow"), "alphabet")
	assert.Truef(t, Contains([]string{"a", "b", "c"}, "c"), "alphabet")
	assert.Falsef(t, Contains([]string{}, "anything"), "empty list")
	assert.Falsef(t, Contains([]string{}, ""), "empty strings")
	assert.Truef(t, Contains([]string{""}, ""), "empty strings")
}

func Test_Unique(t *testing.T) {
	assert.Equal(t, Unique([]string{"a", "b", "c"}), []string{"a", "b", "c"})
	assert.Equal(t, Unique([]string{"a", "a", "a"}), []string{"a"})
	assert.Equal(t, Unique([]string{"a", "b", "a"}), []string{"a", "b"})
}

func Test_ListsContain(t *testing.T) {
	assert.Truef(t, ListsContain("y", []string{"a", "b"}, []string{"x", "y"}), "multiple lists")
	assert.Falsef(t, ListsContain("not present", []string{}, []string{""}), "empty list")
	assert.Falsef(t, ListsContain("not present"), "no lists")
	assert.Truef(t, ListsContain("", []string{"hello", "", "world"}), "empty search term")
}

func Test_Filter(t *testing.T) {
	testFilter := func(elem string) bool { return unicode.IsUpper(rune(elem[0])) }
	in := []string{"Alpha", "beta", "Gamma", "Delta", "epsilon"}
	filtered := Filter(testFilter, in)
	assert.Equal(t, []string{"Alpha", "Gamma", "Delta"}, filtered)

	in2 := []string{"chi", "psi", "Omega"}
	filtered = Filter(testFilter, in, in2)
	assert.Equal(t, []string{"Alpha", "Gamma", "Delta", "Omega"}, filtered)

}

func Test_Difference(t *testing.T) {
	in := []string{"1", "1", "2", "3", "5", "8", "13", "21"}
	sub := []string{"2", "8", "21"}
	correct := []string{"1", "1", "3", "5", "13"}
	assert.Equal(t, correct, Difference(in, sub))
}

func Test_AppendUnique(t *testing.T) {
	// AppendIfUnique is tested via AppendUnique.
	assert.Equal(t,
		[]string{"test"},
		AppendUnique([]string{},
			[]string{"test"}))
	assert.Equal(t,
		[]string{"ab", "cd", "", "ef"},
		AppendUnique([]string{"ab", "cd"},
			[]string{"", "", "ef"}))
	assert.Equal(t,
		[]string{"ab", "cd"},
		AppendUnique([]string{"ab", "cd"},
			[]string{"cd", "ab"}))
}

func Test_Find(t *testing.T) {
	assert.Truef(t, Find([]string{"abc", "abcde"}, "abcd") == -1, "Incorrect index")
	assert.Truef(t, Find([]string{"abc", "abcde"}, "abcde") == 1, "Incorrect index")
}

func Test_Remove(t *testing.T) {
	assert.Equal(t, []string{"abc", "abcde"},
		Remove([]string{"abc", "abcde"}, "abcd"))
	assert.Equal(t, []string{"abc"},
		Remove([]string{"abc", "abcde"}, "abcde"))
	assert.Equal(t, []string{"abcde"},
		Remove([]string{"abc", "abcde"}, "abc"))
}

func Test_Reversed(t *testing.T) {
	assert.Equal(t, []string{},
		Reversed([]string{}))
	assert.Equal(t, []string{""},
		Reversed([]string{""}))
	assert.Equal(t, []string{"234", "123"},
		Reversed([]string{"123", "234"}))
	assert.Equal(t, []string{"..<>", "234", ""},
		Reversed([]string{"", "234", "..<>"}))
}

func Test_StripUnusedArgs(t *testing.T) {
	args := map[string]string{
		"compiler": "gcc",
		"args":     "-Wall -Werror -c",
		"depfile":  "deps.d",
		"wrapper":  "ccache",
		"in":       "source.c",
		"out":      "source.o",
	}
	StripUnusedArgs(args, "${compiler} -o ${out} ${in} ${args}")
	assert.Equal(t, []string{"args", "compiler", "in", "out"}, SortedKeys(args))
}

func Test_Trim(t *testing.T) {
	assert.Equal(t, []string{"hello", "world"},
		Trim([]string{"", " hello ", "world", "	"}))
}

func Test_Join(t *testing.T) {
	assert.Truef(t,
		Join() == "",
		"Empty join should yield an empty string")
	assert.Truef(t,
		Join([]string{"Hello", "world"}) == "Hello world",
		"Didn't concatenate two words")

	// Here, there's a 3rd space before "spaces", because strings.Join()
	// adds one for the empty string, but utils.Join() doesn't.
	assert.Truef(t,
		Join([]string{"this is", " surrounded by "},
			[]string{"", "spaces"}, []string{}) ==
			"this is  surrounded by   spaces",
		"Surrounding space not handled")
}

func Test_NewStringSlice(t *testing.T) {
	// Check problematicAppendExample to understand more what issue we faced.
	// arrA has capacity of exactly the number of elements it's created with
	arrA := []string{"1", "2", "3", "4"} // A = [1 2 3 4] (cap 4)

	// Append one element - the capacity is doubled to cope with future expansion
	arrA = append(arrA, "5") // A = [1 2 3 4 5] (cap 8)

	arrB := NewStringSlice(arrA, []string{"B"})
	// A = [1 2 3 4 5]
	// B = [1 2 3 4 5 B]

	// arrC := append(arrA, "C") <-- problematic usage
	// A = [1 2 3 4 5]
	// B = [1 2 3 4 5 C] // this could be an issue if someone isn't careful
	// C = [1 2 3 4 5 C]

	arrC := NewStringSlice(arrA, []string{"C"})
	// A = [1 2 3 4 5]
	// B = [1 2 3 4 5 B] // as expected
	// C = [1 2 3 4 5 C]
	fmt.Printf("A = %v\n", arrA)
	// B = [1 2 3 4 5 C]
	fmt.Printf("B = %v\n", arrB)
	// C = [1 2 3 4 5 C]
	fmt.Printf("C = %v\n", arrC)
	assert.Equal(t, []string{"1", "2", "3", "4", "5", "C"}, arrC)
	assert.Equal(t, []string{"1", "2", "3", "4", "5", "B"}, arrB)
}

// Below example code with problematic append() call, this is why we have utils.NewStringSlice
func problematicAppendExample(t *testing.T) {
	// arrA has capacity of exactly the number of elements it's created with
	arrA := []string{"1", "2", "3", "4"}

	fmt.Println("----")
	// A = [1 2 3 4] (cap 4)
	fmt.Printf("A = %v (cap %d)\n", arrA, cap(arrA))

	// Append one element - the capacity is doubled to cope with future expansion
	arrA = append(arrA, "5")
	// A = [1 2 3 4 5] (cap 8)
	fmt.Printf("A = %v (cap %d)\n", arrA, cap(arrA))

	arrB := append(arrA, "B")
	fmt.Println("----")
	// A = [1 2 3 4 5]
	fmt.Printf("A = %v\n", arrA)
	// B = [1 2 3 4 5 B]
	fmt.Printf("B = %v\n", arrB)

	arrC := append(arrA, "C")
	fmt.Println("----")
	// A = [1 2 3 4 5]
	fmt.Printf("A = %v\n", arrA)
	// B = [1 2 3 4 5 C]
	fmt.Printf("B = %v\n", arrB)
	// C = [1 2 3 4 5 C]
	fmt.Printf("C = %v\n", arrC)
}

func Test_FlattenPath(t *testing.T) {
	flattened := FlattenPath("a__b/c/d_e/_f_.txt")

	assert.Equal(t, "a__b__c__d_e___f_.txt", flattened)
}

func Test_ExpandReplacesMappedVars(t *testing.T) {
	res := Expand("$$ ${x0}s and ${x1}s ; ${y} $$", func(s string) string {
		dict := map[string]string{
			"x0": "apple",
			"x1": "orange",
			"x2": "carrot",
		}
		if val, ok := dict[s]; ok {
			return val
		} else {
			return "${" + s + "}"
		}
	})

	assert.Equal(t, "$$ apples and oranges ; ${y} $$", res)
}

func Test_ExpandIncomplete(t *testing.T) {
	res := Expand("1 $2 3", func(string) string {
		return ""
	})
	res2 := Expand("1 2 3$", func(string) string {
		return ""
	})
	res3 := Expand("1 2 3${x", func(string) string {
		return ""
	})

	assert.Equal(t, "1 $2 3", res)
	assert.Equal(t, "1 2 3$", res2)
	assert.Equal(t, "1 2 3${x", res3)
}

func Test_SplitPath(t *testing.T) {
	assert.Equal(t, []string{}, SplitPath(""))
	assert.Equal(t, []string{"/"}, SplitPath("/"))
	assert.Equal(t, []string{"/bin"}, SplitPath("/bin"))
	assert.Equal(t, []string{"/bin"}, SplitPath("/bin/"))
	assert.Equal(t, []string{"/usr", "bin"}, SplitPath("/usr/bin"))
	assert.Equal(t, []string{"/usr", "bin"}, SplitPath("/usr/bin/"))
	assert.Equal(t, []string{"rel"}, SplitPath("rel"))
	assert.Equal(t, []string{"a", "rel", "path"}, SplitPath("a/rel/path"))
	assert.Equal(t, []string{"a", "rel", "path"}, SplitPath("a/rel/path/"))
}
