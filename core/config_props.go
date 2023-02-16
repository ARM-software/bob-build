/*
 * Copyright 2018-2021, 2023 Arm Limited.
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

	"github.com/ARM-software/bob-build/internal/utils"
)

type configProperties struct {
	// Map of all available features (e.g. noasserts: { cflags: ["-DNDEBUG]" }),
	// and whether they are enabled or not.
	features map[string]bool

	// Map of all available properties which can be used in templates. Features are
	// not automatically included in this by Bob, so should be added explicitly by the
	// config system if required. These are converted to strings, then made available
	// for use in templates.
	properties map[string]interface{}

	// Sorted array of available features
	featureList []string

	stringMap map[string]string
}

func (properties configProperties) getProp(name string) interface{} {
	if elem, ok := properties.properties[name]; ok {
		return elem
	}
	utils.Die("No property found: %s", name)
	return nil
}

func (properties configProperties) GetBool(name string) bool {
	ret, ok := properties.getProp(name).(bool)
	if ok {
		return ret
	}
	utils.Die("Property %s is not a bool", name)
	return !ret
}

func (properties configProperties) GetInt(name string) int {
	number, ok := properties.getProp(name).(json.Number)
	if !ok {
		utils.Die("Property %s with value '%v' is not an int",
			name, properties.getProp(name))
	}

	ret, err := number.Int64()
	if err != nil {
		utils.Die("Property %s contains invalid int value '%s': %v",
			name, number.String(), err)
	}

	if int64(int(ret)) != ret {
		utils.Die("Property %s value out of `int` range: %d", name, ret)
	}

	return int(ret)
}

func (properties configProperties) GetString(name string) string {
	if ret, ok := properties.getProp(name).(string); ok {
		return ret
	}
	utils.Die("Property %s is not a string", name)
	return ""
}

func (properties configProperties) StringMap() map[string]string {
	return properties.stringMap
}

// This function converts a config value into a string, using the following rules:
//   - booleans are converted into "0" or "1"
//   - Strings are used as-is
//   - Ints are converted into 10-base form
//   - Slices of booleans,strings and ints are converted into a space-separated string
//   - Pointers to booleans,strings and ints are converted into the referenced value
//
// Any other type might Exit().
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
		utils.Die("Can't convert type %s to string!", field.Type().String())
	}
	return value
}

// Identify if the input is a boolean and return its value
func boolValue(thing interface{}) (value, isBool bool) {
	field := reflect.ValueOf(thing)

	switch field.Kind() {
	case reflect.Bool:
		value = field.Bool()
		isBool = true
	default:
		value = false
		isBool = false
	}

	return
}

func (properties *configProperties) LoadConfig(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Unable to read configuration file: %s", err.Error())
	}
	d := json.NewDecoder(bytes.NewReader(content))

	// Decode numbers in JSON as json.Numbers instead of float64.
	// This is actually a string, which is what we want.
	d.UseNumber()

	properties.properties = make(map[string]interface{})
	properties.stringMap = make(map[string]string)
	properties.features = make(map[string]bool)

	var configData map[string]interface{}
	err = d.Decode(&configData)
	if err != nil {
		return fmt.Errorf("Unable to decode json configuration: %s", err.Error())
	}

	for key, val := range configData {

		// get configuration option as a map with interface{}
		configMap := val.(map[string]interface{})

		// Identify that configuration is ignored or not
		if ignore, ok := boolValue(configMap["ignore"]); ok {
			if !ignore {
				properties.properties[key] = configMap["value"]

				// Create a mapping of properties to values that will be used
				// by templates
				properties.stringMap[key] = convertToString(configMap["value"])

				// Identify features and whether they are enabled
				if v, ok := boolValue(configMap["value"]); ok {
					properties.features[key] = v
				}
			}
		}
	}

	// Calculate the plain list of features once.
	properties.featureList = utils.SortedKeysBoolMap(properties.features)

	return nil
}
