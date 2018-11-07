/*
 * Copyright 2018 Arm Limited.
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

package core

import (
	"bytes"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

func applyTemplateString(elem reflect.Value, stringvalues map[string]string, funcmap map[string]interface{}) {
	if elem.Kind() != reflect.String {
		panic("elem is not a string")
	}

	t := template.New("TemplateProps")
	t.Option("missingkey=error")
	t.Funcs(funcmap)

	tmpl, err := t.Parse(elem.String())
	if err != nil {
		panic("Error parsing string '" + elem.String() + "': " + err.Error())
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, stringvalues)
	if err != nil {
		panic("Error executing string '" + elem.String() + "': " + err.Error())
	}
	elem.SetString(buf.String())
}

func applyTemplateRecursive(propsVal reflect.Value, properties *configProperties,
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
			applyTemplateRecursive(field, properties, stringvalues, funcmap)
		}
	}
}

func regMatch(rule string, input string) bool {
	match, _ := regexp.MatchString(rule, input)
	return match
}

func regReplaceString(rule string, input string, replace string) string {
	re := regexp.MustCompile(rule)
	return re.ReplaceAllString(input, replace)
}

// ApplyTemplate writes configuration values (from properties) into the string
// properties in props. This is done recursively.
func ApplyTemplate(props interface{}, properties *configProperties) {
	stringvalues := properties.StringMap()
	funcmap := make(map[string]interface{})
	funcmap["toUpper"] = strings.ToUpper
	funcmap["split"] = strings.Split
	funcmap["match"] = regMatch
	funcmap["replace"] = regReplaceString
	propsVal := reflect.Indirect(reflect.ValueOf(props))

	applyTemplateRecursive(propsVal, properties, stringvalues, funcmap)
}
