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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

type configPropertiesJSON struct {
	// Map of all available features (e.g. noasserts: { cflags: ["-DNDEBUG]" }),
	// and whether they are enabled or not.
	Features map[string]bool
	// Map of all available properties which can be used in templates. Features are
	// not automatically included in this by Bob, so should be added explicitly by the
	// config system if required. These are converted to strings, then made available
	// for use in templates.
	Properties map[string]interface{}
}

type configProperties struct {
	configPropertiesJSON
	stringMap map[string]string
}

func (properties configProperties) getProp(name string) interface{} {
	if elem, ok := properties.Properties[name]; ok {
		return elem
	}
	panic(fmt.Sprintf("No property found: %s", name))
}

func (properties configProperties) GetBool(name string) bool {
	if ret, ok := properties.getProp(name).(bool); ok {
		return ret
	}
	panic(fmt.Sprintf("Property %s is not a bool", name))
}

func (properties configProperties) GetString(name string) string {
	if ret, ok := properties.getProp(name).(string); ok {
		return ret
	}
	panic(fmt.Sprintf("Property %s is not a string", name))
}

func (properties configProperties) StringMap() map[string]string {
	return properties.stringMap
}

func loadConfig(filename string) *configProperties {
	properties := &configProperties{}
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	d := json.NewDecoder(bytes.NewReader(content))

	// Decode numbers in JSON as json.Numbers instead of float64.
	// This is actually a string, which is what we want.
	d.UseNumber()
	err = d.Decode(&properties.configPropertiesJSON)
	if err != nil {
		panic(err)
	}

	properties.stringMap = make(map[string]string)
	for key, val := range properties.Properties {
		properties.stringMap[key] = convertToString(val)
	}

	return properties
}

// This function converts a config value into a string, using the following rules:
//  - booleans are converted into "0" or "1"
//  - Strings are used as-is
//  - Ints are converted into 10-base form
//  - Slices of booleans,strings and ints are converted into a space-separated string
//  - Pointers to booleans,strings and ints are converted into the referenced value
//
// Any other type might panic.
func convertToString(thing interface{}) string {
	field := reflect.ValueOf(thing)
	var value string
	switch field.Kind() {
	case reflect.String:
		value = field.String()

	case reflect.Bool:
		if field.Bool() {
			value = "1"
		} else {
			value = "0"
		}

	case reflect.Int:
		value = strconv.FormatInt(field.Int(), 10)

	case reflect.Ptr:
		if !reflect.Indirect(field).IsValid() {
			// This happens if we have nil pointer. The only time this happens
			// is if we have a "special" boolean.  Ignore these for now.
		} else {
			value = convertToString(reflect.Indirect(field))
		}

	case reflect.Slice:
		values := []string{}
		for j := 0; j < field.Len(); j++ {
			elem := field.Index(j)
			values = append(values, convertToString(elem))
		}
		value = strings.Join(values, " ")

	default:
		panic(fmt.Sprintf("Can't convert type %s to string!", field.Type().String()))
	}
	return value
}
