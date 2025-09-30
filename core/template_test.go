package core

import (
	"testing"

	"github.com/ARM-software/bob-build/core/config"

	"github.com/stretchr/testify/assert"
)

// Create a ConfigProperties type that we can use for testing. This
// avoids having to setup JSON input files for each test.
func setupTestConfig(m map[string]string) *config.Properties {
	properties := &config.Properties{}

	// We can ignore ConfigPropertiesJson, as that is just used to read in the JSON file.
	properties.SetConfig(m)

	return properties
}

type testProperties struct {
	// Normal strings
	StrA string
	StrB string
	StrC string

	// Arrays
	StrArray []string

	// Strings referencing a boolean value
	B1 string
	B2 string

	// Pointer to Strings
	RefA *string
	RefB *string
}

// Check that templates are expanded in a simple property structure
func TestApplyTemplate(t *testing.T) {
	config := setupTestConfig(map[string]string{
		"a": "alpha",
		"b": "beta",
		"c": "gamma",
		"t": "1",
		"f": "0",
	})

	arr := []string{
		"{{.a}}", "{{.b}}", "{{.c}}",
	}
	refA := "{{.a}}{{.t}}"
	refB := "{{.b}}{{.f}}"

	props := testProperties{
		"{{.a}}",
		"{{.b}}",
		"{{.c}}",
		arr,
		"{{.t}}",
		"{{.f}}",
		&refA,
		&refB,
	}
	ApplyTemplate(&props, config)

	// Check templates are expanded in normal strings
	assert.Equalf(t, "alpha", props.StrA, "StrA incorrect")
	assert.Equalf(t, "beta", props.StrB, "StrB incorrect")
	assert.Equalf(t, "gamma", props.StrC, "StrC incorrect")

	// Check 'booleans'. These are actually strings as far as the
	// template code is concerned
	assert.Equalf(t, "1", props.B1, "B1 incorrect")
	assert.Equalf(t, "0", props.B2, "B2 incorrect")

	// Check templates have been expanded in arrays of strings
	assert.Equalf(t, "alpha", props.StrArray[0], "arr[0] incorrect")
	assert.Equalf(t, "beta", props.StrArray[1], "arr[1] incorrect")
	assert.Equalf(t, "gamma", props.StrArray[2], "arr[2] incorrect")

	// Check templates have been expanded in pointers to strings
	assert.Equalf(t, "alpha1", refA, "refA incorrect")
	assert.Equalf(t, "beta0", refB, "refB incorrect")
}

type testNestedProperties struct {
	A, B testProperties
}

// Check that templates are expanded in a nested property structure
func TestApplyTemplateNested(t *testing.T) {
	config := setupTestConfig(map[string]string{
		"a": "alpha",
		"b": "beta",
		"c": "gamma",
		"t": "1",
		"f": "0",
	})

	arr := []string{
		"{{.a}}", "{{.b}}", "{{.c}}",
	}
	arrB := []string{
		"{{.b}}", "{{.c}}", "{{.a}}",
	}
	refA := "{{.a}}{{.t}}"
	refB := "{{.b}}{{.f}}"

	props := testNestedProperties{
		testProperties{
			"{{.a}}",
			"{{.b}}",
			"{{.c}}",
			arr,
			"{{.t}}",
			"{{.f}}",
			&refA,
			&refB,
		},
		testProperties{
			"{{.c}}",
			"{{.a}}",
			"{{.b}}",
			arrB,
			"{{.f}}",
			"{{.t}}",
			nil, // Pointers to string can be nil
			nil,
		},
	}

	ApplyTemplate(&props, config)

	// Check templates are expanded in normal strings
	assert.Equalf(t, "alpha", props.A.StrA, "A.StrA incorrect")
	assert.Equalf(t, "beta", props.A.StrB, "A.StrB incorrect")
	assert.Equalf(t, "gamma", props.A.StrC, "A.StrC incorrect")
	assert.Equalf(t, "gamma", props.B.StrA, "B.StrA incorrect")
	assert.Equalf(t, "alpha", props.B.StrB, "B.StrB incorrect")
	assert.Equalf(t, "beta", props.B.StrC, "B.StrC incorrect")

	// Check 'booleans'. These are actually strings as far as the
	// template code is concerned
	assert.Equalf(t, "1", props.A.B1, "A.B1 incorrect")
	assert.Equalf(t, "0", props.A.B2, "A.B2 incorrect")
	assert.Equalf(t, "0", props.B.B1, "B.B1 incorrect")
	assert.Equalf(t, "1", props.B.B2, "B.B2 incorrect")

	// Check templates have been expanded in arrays of strings
	assert.Equalf(t, "alpha", props.A.StrArray[0], "arr[0] incorrect")
	assert.Equalf(t, "beta", props.A.StrArray[1], "arr[1] incorrect")
	assert.Equalf(t, "gamma", props.A.StrArray[2], "arr[2] incorrect")
	assert.Equalf(t, "beta", props.B.StrArray[0], "arrB[0] incorrect")
	assert.Equalf(t, "gamma", props.B.StrArray[1], "arrB[1] incorrect")
	assert.Equalf(t, "alpha", props.B.StrArray[2], "arrB[2] incorrect")

	// Check templates have been expanded in pointers to strings
	assert.Equalf(t, "alpha1", refA, "refA incorrect")
	assert.Equalf(t, "beta0", refB, "refB incorrect")
}

func TestShellExpressionSplitter(t *testing.T) {
	a := "-DFOO -DBAR"
	refA := []string{
		"-DFOO",
		"-DBAR",
	}
	b := "-DFOO=\"BAR ZETA\" -DOTHER"
	refB := []string{
		"-DFOO=BAR ZETA",
		"-DOTHER",
	}

	c := "-Wl,-rpath=otherthing -DFOO"
	refC := []string{
		"-Wl,-rpath=otherthing",
		"-DFOO",
	}

	d := "-Wl,-rpath=otherthing -DFOO -Wl,-rpath=\"with space\""
	refD := []string{
		"-Wl,-rpath=otherthing",
		"-DFOO",
		"-Wl,-rpath=with space",
	}

	assert.Equalf(t, refA, SplitShell(a), "refA incorrect")
	assert.Equalf(t, refB, SplitShell(b), "refB incorrect")
	assert.Equalf(t, refC, SplitShell(c), "refC incorrect")
	assert.Equalf(t, refD, SplitShell(d), "refD incorrect")
}

type TestShlexProps struct {
	A []string
	B []string
}

func TestShlexExpansion(t *testing.T) {
	config := setupTestConfig(map[string]string{
		"flags1": "-DFOO -DBAR=\"some value\"",
		"flags2": "-Wl,-rpath=otherthing -DFOO -Wl,-rpath=\"with space\"",
	})
	arrA := []string{
		"{{shlex .flags1}}",
		"-DFLAG",
	}
	refA := []string{
		"-DFOO",
		"-DBAR=\"some value\"",
		"-DFLAG",
	}
	arrB := []string{
		"{{shlex .flags2}}",
		"-DFLAG",
	}
	refB := []string{
		"-Wl,-rpath=otherthing",
		"-DFOO",
		"-Wl,-rpath=\"with space\"",
		"-DFLAG",
	}

	props := TestShlexProps{
		arrA,
		arrB,
	}

	ApplyTemplate(&props, config)
	assert.Equalf(t, refA, props.A, "A incorrect")
	assert.Equalf(t, refB, props.B, "B incorrect")
}
