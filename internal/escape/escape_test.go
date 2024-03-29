package escape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name string
	in   string
	out  string
}

var makefileEscapeTests = []testCase{
	{
		name: "no escaping",
		in:   `sed -e "s/a/b/g" input.txt > output.txt`,
		out:  `sed -e "s/a/b/g" input.txt > output.txt`,
	},
	{
		name: "leading $", // typically shell environment expansions
		in:   "$PATH",
		out:  "$$PATH",
	},
	{
		name: "trailing $", // atypical case. ensure we get last char.
		in:   "trailing$",
		out:  "trailing$$",
	},
	{
		name: "multiple $",
		in:   "PATH=$PATH LD_LIBRARY_PATH=$LD_LIBRARY_PATH",
		out:  "PATH=$$PATH LD_LIBRARY_PATH=$$LD_LIBRARY_PATH",
	},
}

func TestMakefileEscaping(t *testing.T) {
	for _, testcase := range makefileEscapeTests {
		out := MakefileEscape(testcase.in)
		assert.Equalf(t, out, testcase.out, "Test case %s",
			testcase.name)
	}
}

var templatedStringTests = []testCase{
	{
		name: "no template, no escape",
		in:   "The quick brown fox",
		out:  "The quick brown fox",
	},
	{
		name: "only template",
		in:   "{{function .param \"$tring\" 5}}",
		out:  "{{function .param \"$tring\" 5}}",
	},
	{
		name: "escape before template",
		in:   "The $QUICK ${{color 4 \"$\"}} fox",
		out:  "The $$QUICK $${{color 4 \"$\"}} fox",
	},
	{
		name: "escape after template",
		in:   "The {{adjective 3 \"$\"}}$BROWN $FOX",
		out:  "The {{adjective 3 \"$\"}}$$BROWN $$FOX",
	},
	{
		name: "partial template at start",
		in:   "The $quick}} {{brown \"$\"}} fox",
		out:  "The $$quick}} {{brown \"$\"}} fox",
	},
	{
		name: "partial template at end",
		in:   "The {{quick \"$\"}} {{$brown fox",
		out:  "The {{quick \"$\"}} {{$$brown fox",
	},
	{
		name: "Interleaved templates",
		in:   "The $QUICK {{adjective 4 \"$\"}} $BROWN {{mammal 3 \"$\"}}",
		out:  "The $$QUICK {{adjective 4 \"$\"}} $$BROWN {{mammal 3 \"$\"}}",
	},
	{
		name: "Adjacent templates",
		in:   "The $QUICK {{adjective 4 \"$\"}}{{color 4 \"$\"}} $MAMMAL",
		out:  "The $$QUICK {{adjective 4 \"$\"}}{{color 4 \"$\"}} $$MAMMAL",
	},
	{
		name: "Unmatched }}",
		in:   "The $QUICK {{adjective 4 \"$\"}} color 4 \"$\"}} {{mammal 3 \"$\"}}",
		out:  "The $$QUICK {{adjective 4 \"$\"}} color 4 \"$$\"}} {{mammal 3 \"$\"}}",
	},
}

func TestEscapeTemplatedString(t *testing.T) {
	for _, testcase := range templatedStringTests {
		out := EscapeTemplatedString(testcase.in, MakefileEscape)
		assert.Equalf(t, out, testcase.out, "Test case %s",
			testcase.name)
	}
}
