package core

import (
	"bytes"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/internal/utils"
)

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

func applyTemplateRecursive(propsVal reflect.Value,
	stringvalues map[string]string, funcmap map[string]interface{}) {

	for i := 0; i < propsVal.NumField(); i++ {
		field := propsVal.Field(i)

		switch field.Kind() {
		case reflect.String:
			applyTemplateString(field, stringvalues, funcmap)

		case reflect.Slice:
			// Array of strings
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
