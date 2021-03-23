/*
 * Copyright 2018-2019, 2021 Arm Limited.
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
)

// featurePropertyName returns name of feature. Name needs to start from capital letter because
// this is how it works in go exported/unexported properties
// e.g. Android, Foo_bar
func featurePropertyName(name string) string {
	result := strings.ToLower(name) // e.g. android, foo_bar
	if len(name) > 0 {
		return strings.ToUpper(name[0:1]) + result[1:]
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

type singleFeature struct {
	BlueprintEmbed interface{}
}

func typesOf(list ...interface{}) []reflect.Type {
	types := make([]reflect.Type, len(list))
	for i, element := range list {
		types[i] = reflect.TypeOf(element)
	}
	return types
}

// Init generates and initializes a struct containing a field of type
// 'propsType' for every available feature. 'propsType' will be constructed
// from list of types. By constructed we mean properties of each
// type will be merged together. It is important to set here
// every available feature not only enabled ones, because blueprint will
// fail during reading .bp files. The generated object is embedded
// in each module types' properties instance, and is used by Blueprint to
// decide what properties can be set inside features in each module type.
//
// An example generated type:
// type BlueprintEmbedType struct {
//         Debug PropsType
//         Enable_something PropsType
//         Some_other_feature PropsType
// }
// Name of each property in this struct is custom feature name.
// Blueprint will inflate this structure with data read from .bp files.
// Only exported properties can be set so property name MUST start from capital letter.
func (f *Features) Init(properties *configProperties, list ...interface{}) {
	if len(list) == 0 {
		panic("List can't be empty")
	}

	propsType := coalesceTypes(typesOf(list...)...)
	fields := make([]reflect.StructField, len(properties.featureList))

	for i, featureName := range properties.featureList {
		fields[i] = reflect.StructField{
			Name: featurePropertyName(featureName),
			Type: reflect.TypeOf(singleFeature{}),
		}
	}

	bpFeatureStruct := reflect.StructOf(fields)
	instancePtr := reflect.New(bpFeatureStruct)
	f.BlueprintEmbed = instancePtr.Interface()

	instance := reflect.Indirect(instancePtr)
	for i := range properties.featureList {
		propsInFeature := instance.Field(i).Addr().Interface().(*singleFeature)
		propsInFeature.BlueprintEmbed = reflect.New(propsType).Interface()
	}

}

// coalesceTypes will squash multiple types to new type. This has different result
// than Go composition of structs.
//
// Example (go composition):
// type compositeStruct struct {
//     testPropsGroupA
//     testPropsGroupB
// }
// Debug print:
// {
//     testPropsGroupA: core.testPropsGroupA
//     {
//       Field_a: string
//       Field_c: string
//       Field_f: string
//     }
//     testPropsGroupB: core.testPropsGroupB
//     {
//       Field_b: string
//     }
// }
//
// Example for: coalesceTypes([]reflect.Type{
//    testPropsGroupA{},
//    testPropsGroupB{},
// })
// Debug print:
// {
//     Field_a: string
//     Field_c: string
//     Field_f: string
//     Field_b: string
// }
func coalesceTypes(list ...reflect.Type) reflect.Type {
	if len(list) == 0 {
		panic("List can't be empty")
	}
	if len(list) == 1 {
		return list[0]
	}

	fieldsKeys := map[string]bool{}
	fields := []reflect.StructField{}

	for _, elementType := range list {
		for i := 0; i < elementType.NumField(); i++ {
			field := elementType.Field(i)
			fieldName := field.Name
			if _, ok := fieldsKeys[fieldName]; ok {
				panic(fmt.Sprintf("Name collision: '%v'\n", fieldName))
			} else {
				fieldsKeys[fieldName] = true
				fields = append(fields, field)
			}
		}
	}

	return reflect.StructOf(fields)
}

// AppendProps merges properties from BlueprintEmbed to dst, but only for enabled features
// expect that Features are inited (before using this function we should call Features.Init)
// expect that properties.Features should contain all available features (whenever disabled/enabled)
func (f *Features) AppendProps(dst []interface{}, properties *configProperties) error {
	// featuresData is struct created in Features.Init function
	featuresData := reflect.ValueOf(f.BlueprintEmbed).Elem()

	for _, featureKey := range properties.featureList {
		if properties.features[featureKey] { // Check the feature is enabled
			// Features are matched like "Feature_name" - feature structure
			featureFieldName := featurePropertyName(featureKey)
			featureStruct := featuresData.FieldByName(featureFieldName)
			if !featureStruct.IsValid() {
				panic(fmt.Sprintf("Field returned for property %s isn't valid\n", featureFieldName))
			}
			// AppendProperties expects a pointer to a struct.
			featureStructPointer := featureStruct.FieldByName("BlueprintEmbed").Interface()

			// If featureProps is nil then we've determined that we can skip this,
			// so avoid calling AppendProperties
			if featureStructPointer != nil {
				err := AppendMatchingProperties(dst, featureStructPointer)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
