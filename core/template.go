package core

import (
	"bytes"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"unicode"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/internal/utils"
)

func splitShell(s string) []string {
	var out []string
	var buf []rune
	inQuotes := false
	escape := false

	flush := func() {
		if len(buf) > 0 {
			out = append(out, string(buf))
			buf = buf[:0]
		}
	}

	for _, r := range s {
		if escape {
			buf = append(buf, r)
			escape = false
			continue
		}
		switch r {
		case '\\':
			escape = true
		case '"':
			inQuotes = !inQuotes
		default:
			if !inQuotes && unicode.IsSpace(r) {
				flush()
			} else {
				buf = append(buf, r)
			}
		}
	}
	flush()
	return out
}

func applyTemplateString(elem reflect.Value, stringvalues map[string]string, funcmap map[string]interface{}) {
	if elem.Kind() != reflect.String {
		utils.Die("elem is not a string")
	}

	t := template.New("TemplateProps")
	t.Option("missingkey=error")
	t.Funcs(funcmap)

	tmpl, err := t.Parse(elem.String())
	if err != nil {
		utils.Die("Error parsing string '%s': %s", elem.String(), err.Error())
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, stringvalues)
	if err != nil {
		utils.Die("Error executing string '%s': %s", elem.String(), err.Error())
	}
	elem.SetString(buf.String())
}

// When processing a slice and expanding templates, we won't to
// specifically not process untemplated strings as templated
// strings will under-go a shlex split
func shlexExpand(field reflect.Value, stringvalues map[string]string) {
	// matches a value in the format of {{shlex .<value>}}
	pattern := "^\\{\\{\\s*shlex\\s+\\.(\\w+)\\s*\\}\\}$"
	regexpr := regexp.MustCompile(pattern)
	if field.Len() < 1 {
		return
	}
	if field.Index(0).Kind() != reflect.String {
		return
	}
	var newSlice []string = make([]string, 0)
	for j := 0; j < field.Len(); j++ {
		elem := field.Index(j)
		match := regexpr.MatchString(elem.String())
		if !match {
			newSlice = append(newSlice, elem.String())
			continue
		}
		captures := regexpr.FindStringSubmatch(elem.String())
		// Capture group is always first index. It has to exist since we have a match
		key := strings.TrimLeft(captures[1], ".")
		val := stringvalues[key]
		escaped := strings.ReplaceAll(val, "\"", "\"\\\"")
		split := splitShell(escaped)
		newSlice = append(newSlice, split...)
	}

	field.Set(reflect.ValueOf(newSlice))
}

func applyTemplateRecursive(propsVal reflect.Value,
	stringvalues map[string]string, funcmap map[string]interface{}) {

	for i := 0; i < propsVal.NumField(); i++ {
		field := propsVal.Field(i)

		switch field.Kind() {
		case reflect.String:
			applyTemplateString(field, stringvalues, funcmap)

		case reflect.Slice:
			// Array of strings
			shlexExpand(field, stringvalues)
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if elem.Kind() == reflect.String {
					applyTemplateString(elem, stringvalues, funcmap)
				}
			}

		case reflect.Ptr:
			tgtField := reflect.Indirect(field)
			if tgtField.Kind() == reflect.String {
				applyTemplateString(tgtField, stringvalues, funcmap)
			}

		case reflect.Struct:
			applyTemplateRecursive(field, stringvalues, funcmap)
		}
	}
}

func regMatch(rule string, input string) bool {
	match, _ := regexp.MatchString(rule, input)
	return match
}

func regReplace(rule string, input string, replace string) string {
	re := regexp.MustCompile(rule)
	return re.ReplaceAllString(input, replace)
}

func matchSrcs(input string) string {
	return "{{match_srcs \"" + input + "\"}}"
}

func filter_compiler_flags(flag string) string {
	return "{{add_if_supported \"" + flag + "\"}}"
}

// ApplyTemplate writes configuration values (from properties) into the string
// properties in props. This is done recursively.
func ApplyTemplate(props interface{}, properties *config.Properties) {
	stringvalues := properties.StringMap()
	funcmap := make(map[string]interface{})
	funcmap["to_upper"] = strings.ToUpper
	funcmap["to_lower"] = strings.ToLower
	funcmap["split"] = strings.Split
	funcmap["reg_match"] = regMatch
	funcmap["reg_replace"] = regReplace
	funcmap["match_srcs"] = matchSrcs
	funcmap["add_if_supported"] = filter_compiler_flags
	propsVal := reflect.Indirect(reflect.ValueOf(props))
	applyTemplateRecursive(propsVal, stringvalues, funcmap)
}
