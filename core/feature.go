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
	"fmt"
	"reflect"
	"strings"

	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/utils"
)

func titleFirst(a string) string {
	result := ""
	if len(a) > 0 {
		result += strings.ToUpper(a[0:1])
	}
	if len(a) > 1 {
		result += strings.ToLower(a[1:])
	}
	return result
}

// Features must be embedded in each modules property structure to support the
// use of features in the module. The feature must be initialised with a call
// to Init().
type Features struct {
	// 'BlueprintEmbed' is a special case in Blueprint which makes it interpret
	// a runtime-generated type as being embedded in its parent struct.
	BlueprintEmbed interface{}
}

// Init generates and initializes a struct containing a field of type
// 'propsType' for every available feature. The generated object is embedded
// in each module types' properties instance, and is used by Blueprint to
// decide what properties can be set inside features in each module type.
//
// An example generated type:
// type featureSetType struct {
//         Debug PropsType
//         Enable_something PropsType
//         Some_other_feature PropsType
// }
func (f *Features) Init(availableFeatures []string, propsType reflect.Type) {
	fields := make([]reflect.StructField, len(availableFeatures))

	for i, key := range availableFeatures {
		field := reflect.StructField{
			Name: titleFirst(key),
			Type: propsType,
		}
		fields[i] = field
	}

	featureSetType := reflect.StructOf(fields)
	featureSetValPtr := reflect.New(featureSetType)
	f.BlueprintEmbed = featureSetValPtr.Interface()
}

// AppendProps merges properties from BlueprintEmbed to dst, but only for enabled features
func (f *Features) AppendProps(dst []interface{}, properties *configProperties) error {
	featureSetVal := reflect.ValueOf(f.BlueprintEmbed).Elem()

	for _, key := range utils.SortedKeysBoolMap(properties.Features) {
		// Check the feature is enabled
		if properties.Features[key] {
			field := featureSetVal.FieldByName(titleFirst(key))

			if !field.IsValid() {
				panic(fmt.Sprintf("Field returned for property %s isn't valid\n", key))
			}

			// AppendProperties expects a pointer to a struct.
			featureProps := field.Addr().Interface()

			// If featureProps is nil then we've determined that we can skip this,
			// so avoid calling AppendProperties
			if featureProps != nil {
				err := proptools.AppendMatchingProperties(dst, featureProps, nil)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
