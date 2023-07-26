package escape

import (
	"strings"

	"github.com/google/blueprint/proptools"
)

var makefileEscaper = strings.NewReplacer("$", "$$")

// Escape characters in a string that Make may interpret in recipes
// (as part of the rule context).
//
// Ocurrences of $ need to be escaped as $$. The new escaped string is
// returned.
func MakefileEscape(s string) string {
	return makefileEscaper.Replace(s)
}

// Escape characters in a list of strings that Make may interpret in
// recipes (as part of the rule context).
//
// A new slice containing the escaped strings is returned.
func MakefileEscapeList(list []string) []string {
	// Create a new slice initialised with the initial list
	list = append([]string(nil), list...)
	for i, s := range list {
		list[i] = MakefileEscape(s)
	}
	return list
}

// Escape characters that are special to either Make or the shell.
//
// The new escaped string is returned.
func MakefileAndShellEscape(s string) string {
	return proptools.ShellEscape(MakefileEscape(s))
}

// Escape characters that are special to either Make or the shell.
//
// A new slice containing the escaped strings is returned.
func MakefileAndShellEscapeList(list []string) []string {
	return proptools.ShellEscapeList(MakefileEscapeList(list))
}

// Escape a string which may contain Go templates.
//
// The content of the template is not escaped.
//
// This function is only useful where the template output is not
// expected to be escaped. If the template output needs escaping too,
// then expand the template before escaping instead.
//
// The new escaped string is returned.
func EscapeTemplatedString(s string, escapeFn func(string) string) string {
	var str strings.Builder

	for len(s) > 0 {
		start := strings.Index(s, "{{")
		end := strings.Index(s, "}}")
		if start >= 0 && end > start {
			end += 2
			str.WriteString(escapeFn(s[0:start]))
			str.WriteString(s[start:end])
			s = s[end:]
		} else if end >= 0 && start > end {
			// Ignore closing }} without a matching {{
			// These will be escaped if needed.
			str.WriteString(escapeFn(s[0:start]))
			s = s[start:]
		} else {
			// No further template, or unclosed final
			str.WriteString(escapeFn(s))
			s = ""
		}
	}

	return str.String()
}

// Escape a string which may contain Go templates.
//
// A new slice containing the escaped strings is returned.
func EscapeTemplatedStringList(list []string, escapeFn func(string) string) []string {
	// Create a new slice initialised with the initial list
	list = append([]string(nil), list...)
	for i, s := range list {
		list[i] = EscapeTemplatedString(s, escapeFn)
	}
	return list
}
