package warnings

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureStderr(f func()) string {
	r, w, err := os.Pipe()

	if err != nil {
		panic(err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	f()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()

}

func TestWarningDefault(t *testing.T) {
	const expected string = "BpFile,BpModule,WarningAction,WarningMessage,WarningCategory\n" +
		"A/build.bp,gen_table,ignore,Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead.,relative-up-link\n"
	var msg strings.Builder

	wr := New(&msg, "")
	wr.Warn(RelativeUpLinkWarning, "A/build.bp", "gen_table")

	assert.Equal(t, expected, msg.String())
}

func TestWarning(t *testing.T) {
	const expected string = "BpFile,BpModule,WarningAction,WarningMessage,WarningCategory\n" +
		"A/build.bp,gen_table,warning,`bob_generate_source` should not be used. Use `bob_genrule` instead.,generate-rule\n" +
		"B/build.bp,gen_binary,ignore,Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead.,relative-up-link\n"

	const expectedStderr string = "A/build.bp:gen_table: warning: `bob_generate_source` should not be used. Use `bob_genrule` instead. [generate-rule]\n"
	var msg strings.Builder

	wr := New(&msg, "GenerateRuleWarning:W")

	stderrOutput := captureStderr(func() {
		wr.Warn(GenerateRuleWarning, "A/build.bp", "gen_table")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary")
	})

	assert.Equal(t, expected, msg.String())
	assert.Equal(t, expectedStderr, stderrOutput)
}

func TestErrorWarning(t *testing.T) {
	const expected string = "BpFile,BpModule,WarningAction,WarningMessage,WarningCategory\n" +
		"A/build.bp,gen_table,error,`bob_generate_source` should not be used. Use `bob_genrule` instead.,generate-rule\n" +
		"B/build.bp,gen_binary,warning,Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead.,relative-up-link\n"

	const expectedStderr string = "A/build.bp:gen_table: error: `bob_generate_source` should not be used. Use `bob_genrule` instead. [generate-rule]\n" +
		"B/build.bp:gen_binary: warning: Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead. [relative-up-link]\n"

	var msg strings.Builder

	wr := New(&msg, "GenerateRuleWarning:E RelativeUpLinkWarning:W")
	stderrOutput := captureStderr(func() {
		wr.Warn(GenerateRuleWarning, "A/build.bp", "gen_table")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary")
	})

	assert.Equal(t, expected, msg.String())
	assert.Equal(t, expectedStderr, stderrOutput)
}

func TestFilterOverwriteCategory(t *testing.T) {
	const expected string = "Overriding warning category not allowed: 'GenerateRuleWarning:W'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "GenerateRuleWarning:E RelativeUpLinkWarning:W GenerateRuleWarning:W")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestFilterOverwriteWildcard(t *testing.T) {
	const expected string = "Overriding wildcard (*) not allowed: '*:W'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "*:E RelativeUpLinkWarning:W *:W")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestErrorAllWarnings(t *testing.T) {
	const expectedStderr string = "A/build.bp:gen_table: error: `bob_generate_source` should not be used. Use `bob_genrule` instead. [generate-rule]\n" +
		"B/build.bp:gen_binary: warning: Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead. [relative-up-link]\n"

	const expected string = "BpFile,BpModule,WarningAction,WarningMessage,WarningCategory\n" +
		"A/build.bp,gen_table,error,`bob_generate_source` should not be used. Use `bob_genrule` instead.,generate-rule\n" +
		"B/build.bp,gen_binary,warning,Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead.,relative-up-link\n"

	var msg strings.Builder

	wr := New(&msg, "*:E RelativeUpLinkWarning:W")
	stderrOutput := captureStderr(func() {
		wr.Warn(GenerateRuleWarning, "A/build.bp", "gen_table")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary")
	})

	assert.Equal(t, expected, msg.String())
	assert.Equal(t, expectedStderr, stderrOutput)
}

func TestWrongFilterCategory(t *testing.T) {
	const expected string = "Wrong filter category 'Ridiculous:E'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "Ridiculous:E")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestWrongFilterAction(t *testing.T) {
	const expected string = "Wrong filter action 'DirectPathsWarning:D'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "DirectPathsWarning:D")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestWrongFilter(t *testing.T) {
	const expected string = "Wrong warnings filter expression 'DirectPathsWarning'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "DirectPathsWarning RelativeUpLinkWarning:W")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestCheckIfErrors(t *testing.T) {
	var msg strings.Builder

	wr := New(&msg, "GenerateRuleWarning:E RelativeUpLinkWarning:W")
	captureStderr(func() {
		wr.Warn(GenerateRuleWarning, "A/build.bp", "gen_table")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary")
		assert.Equal(t, 1, wr.ErrorWarnings())
		wr.Warn(GenerateRuleWarning, "ABC/build.bp", "gen_lib")
		wr.Warn(RelativeUpLinkWarning, "BCD/build.bp", "gen_binary_two")
		assert.Equal(t, 2, wr.ErrorWarnings())
	})
}

func TestHyperlinks(t *testing.T) {
	const expectedHyperlink string = "A/build.bp:gen_table: error: `bob_generate_source` should not be used. Use `bob_genrule` instead. " +
		"[\x1b]8;;https://github.com/ARM-software/bob-build/tree/master/docs/warnings/generate-rule.md\agenerate-rule\x1b]8;;\a]\n" +
		"B/build.bp:gen_binary: warning: Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead. [\x1b]8;;" +
		"https://github.com/ARM-software/bob-build/tree/master/docs/warnings/relative-up-link.md\arelative-up-link\x1b]8;;\a]\n"

	const expected string = "A/build.bp:gen_table: error: `bob_generate_source` should not be used. Use `bob_genrule` instead. [generate-rule]\n" +
		"B/build.bp:gen_binary: warning: Relative up-links in `srcs` are not allowed. Use `bob_filegroup` instead. [relative-up-link]\n"

	var msg strings.Builder

	type Tuple struct {
		a, b, exp interface{}
	}

	config := [4]Tuple{
		{"DOMTERM", "DOMTERM_PRESENT", expectedHyperlink},
		{"VTE_VERSION", "6003", expectedHyperlink},
		{"VTE_VERSION", "3405", expected},
		{"TERM", "xterm-256color", expectedHyperlink},
	}

	var wr *WarningLogger

	for _, item := range config {
		os.Setenv(item.a.(string), item.b.(string))
		wr = New(&msg, "GenerateRuleWarning:E RelativeUpLinkWarning:W")

		stderrOutput := captureStderr(func() {
			wr.Warn(GenerateRuleWarning, "A/build.bp", "gen_table")
			wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary")
			assert.Equal(t, 1, wr.ErrorWarnings())
		})

		assert.Equal(t, item.exp.(string), stderrOutput)

		os.Unsetenv(item.a.(string))
	}
}

func TestInfoMessage(t *testing.T) {
	const expectedHyperlink string = "For more information on Bob warnings, see: [\x1b]8;;https://github.com/ARM-software/bob-build/" +
		"tree/master/docs/warnings/warnings.md\ahttps://github.com/ARM-software/bob-build/tree/master/docs/warnings/warnings.md\x1b]8;;\a]"

	const expected string = "For more information on Bob warnings, see: " +
		"[https://github.com/ARM-software/bob-build/tree/master/docs/warnings/warnings.md]"

	os.Setenv("TERM", "xterm")

	var msg strings.Builder

	wr := New(&msg, "*:E")
	str := wr.InfoMessage()

	assert.Equal(t, expectedHyperlink, str)

	os.Unsetenv("TERM")

	wr = New(&msg, "*:W")
	str = wr.InfoMessage()

	assert.Equal(t, expected, str)
}
