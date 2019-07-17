/*
 * Copyright 2018-2019 Arm Limited.
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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ARM-software/bob-build/utils"
)

// enabledFeatures is just wrapper function to easily enable features that we want
func enabledFeatures(featuresList ...string) (properties configProperties) {
	properties.features = make(map[string]bool)
	for _, feature := range featuresList {
		// Features keys should be in lowercase
		properties.features[feature] = true
	}
	properties.featureList = utils.SortedKeysBoolMap(properties.features)
	return
}

func printDebug(value reflect.Value) {
	printDebugType(value.Type())
}

func printDebugType(value reflect.Type, optionalIndent ...int) {
	indent := 0
	if len(optionalIndent) > 0 {
		indent = optionalIndent[0]
	}
	const indention = "\t"
	fmt.Printf("%s{\n", strings.Repeat(indention, indent))
	indent++
	for i := 0; i < value.NumField(); i++ {
		fmt.Printf("%s%v: %v\n", strings.Repeat(indention, indent), value.Field(i).Name, value.Field(i).Type)
		if value.Field(i).Type.Kind() == reflect.Struct {
			printDebugType(value.Field(i).Type, indent)
		}
	}
	indent--
	fmt.Printf("%s}\n", strings.Repeat(indention, indent))
}

// injectData will 'inject' 'data' to Features.BlueprintEmbed
// runtime structure. It behaves like injecting json values
// to matching keys e.g. level1.level2.variable.
// Check printDebug output to more easily navigate.
func (features *Features) injectData(featureName string, path string, data interface{}) {
	allFeatures := reflect.ValueOf(features.BlueprintEmbed).Elem()
	if !allFeatures.IsValid() {
		printDebug(reflect.ValueOf(features.BlueprintEmbed).Elem())
		panic(fmt.Sprintf("invalid '%s'\n", path))
	}

	propsInFeatureVal := allFeatures.FieldByName(featureName)
	if !propsInFeatureVal.IsValid() {
		printDebug(reflect.ValueOf(allFeatures))
		panic(fmt.Sprintf("Couldn't find struct for feature '%s'", featureName))
	}
	propsInFeature := propsInFeatureVal.Interface().(singleFeature)

	value := reflect.ValueOf(propsInFeature.BlueprintEmbed).Elem()

	for _, name := range strings.Split(path, ".") {
		previous := value
		value = value.FieldByName(name)
		if !value.IsValid() {
			printDebug(previous)
			panic(fmt.Sprintf("invalid '%s' in '%s'\n", name, path))
		}
	}
	value.Set(reflect.ValueOf(data)) // final field
}

type testProps struct {
	FieldA string
	FieldB string
	FieldC string
	FieldD string // features don't have to correspond 1:1
	FieldE string
	FieldF string
	FieldG string

	Features // containing COPY of ALL features filled in using reflection (normally done by blueprint)
}

type testPropsGroupA struct {
	FieldA string
	FieldC string
	FieldG string
}
type testPropsGroupB struct {
	FieldB string
}
type testPropsGroupC struct {
	FieldE string
	FieldF string
}

func createTestModuleAndFeatures() (testProps, configProperties) {
	module := testProps{
		FieldA: "a",
		FieldB: "b",
		FieldC: "c",
		FieldD: "d",
		FieldE: "e",
		FieldF: "f",
		FieldG: "g",
	}

	featuresNames := []string{
		"feature_a",
		"feature_b",
		"feature_c",
		"feature_d",
	}

	properties := enabledFeatures(featuresNames...)

	module.Init(&properties,
		testPropsGroupA{},
		testPropsGroupB{},
		testPropsGroupC{},
	)

	module.injectData("Feature_a", "FieldA", "Props_a")
	module.injectData("Feature_a", "FieldC", "Props_c")
	module.injectData("Feature_a", "FieldG", "Props_g")
	module.injectData("Feature_b", "FieldB", "Props_b")
	module.injectData("Feature_c", "FieldE", "Props_e")
	module.injectData("Feature_c", "FieldF", "Props_f")
	module.injectData("Feature_d", "FieldA", "+D_a")
	module.injectData("Feature_d", "FieldC", "+D_c")
	module.injectData("Feature_d", "FieldG", "+D_g")
	module.injectData("Feature_d", "FieldB", "+D_b")
	module.injectData("Feature_d", "FieldE", "+D_e")
	module.injectData("Feature_d", "FieldF", "+D_f")

	return module, properties
}

func Test_should_return_expected_default_values_when_using_setup_function(t *testing.T) {
	module, properties := createTestModuleAndFeatures()

	assert.True(t, properties.features["feature_a"], "feature_a should be enabled by default")
	assert.True(t, properties.features["feature_b"], "feature_b should be enabled by default")
	assert.True(t, properties.features["feature_c"], "feature_c should be enabled by default")
	assert.True(t, properties.features["feature_d"], "feature_d should be enabled by default")

	assert.Equalf(t, "a", module.FieldA, "module.FieldA should be equal to default value")
	assert.Equalf(t, "b", module.FieldB, "module.FieldB should be equal to default value")
	assert.Equalf(t, "c", module.FieldC, "module.FieldC should be equal to default value")
	assert.Equalf(t, "d", module.FieldD, "module.FieldD should be equal to default value")
	assert.Equalf(t, "e", module.FieldE, "module.FieldE should be equal to default value")
	assert.Equalf(t, "f", module.FieldF, "module.FieldF should be equal to default value")
	assert.Equalf(t, "g", module.FieldG, "module.FieldG should be equal to default value")
}

func Test_should_not_change_when_appending_empty_features(t *testing.T) {
	module, properties := createTestModuleAndFeatures()

	// BlueprintEmbed must be inited! So BlueprintEmbed can't be nil!
	module.Init( // Just make new Init so we will have "empty structure"
		&properties,
		testPropsGroupA{},
		testPropsGroupB{},
		testPropsGroupC{},
	)

	if err := module.AppendProps([]interface{}{&module}, &properties); err != nil {
		panic(err)
	}
}

func Test_should_append_matching_properties_when_one_feature_is_enabled(t *testing.T) {
	module, properties := createTestModuleAndFeatures()
	properties.features["feature_a"] = false
	properties.features["feature_c"] = false
	properties.features["feature_d"] = false

	assert.True(t, properties.features["feature_b"], "Feature should be enabled")
	if err := module.AppendProps([]interface{}{&module}, &properties); err != nil {
		panic(err)
	}

	assert.Equalf(t, "a", module.FieldA, "module.FieldA incorrect")
	assert.Equalf(t, "bProps_b", module.FieldB, "module.FieldB incorrect")
	assert.Equalf(t, "c", module.FieldC, "module.FieldC incorrect")
	assert.Equalf(t, "d", module.FieldD, "module.FieldD can't be changed") // No feature has this property
	assert.Equalf(t, "e", module.FieldE, "module.FieldE incorrect")
	assert.Equalf(t, "f", module.FieldF, "module.FieldF incorrect")
	assert.Equalf(t, "g", module.FieldG, "module.FieldG incorrect")
}

func Test_should_not_modify_when_no_feature_is_enabled(t *testing.T) {
	module, properties := createTestModuleAndFeatures()
	// The current implementation allows for properties.Features to contain
	// a subset of the features Init was called with - check that this works.
	// However, this functionality is not actually required by Bob.
	properties.features = map[string]bool{} // all disabled (when key isn't present it should be treated as disabled)

	if err := module.AppendProps([]interface{}{&module}, &properties); err != nil {
		panic(err)
	}

	assert.Equalf(t, "a", module.FieldA, "module.FieldA incorrect")
	assert.Equalf(t, "b", module.FieldB, "module.FieldB incorrect")
	assert.Equalf(t, "c", module.FieldC, "module.FieldC incorrect")
	assert.Equalf(t, "d", module.FieldD, "module.FieldD can't be changed") // No feature has this property
	assert.Equalf(t, "e", module.FieldE, "module.FieldE incorrect")
	assert.Equalf(t, "f", module.FieldF, "module.FieldF incorrect")
	assert.Equalf(t, "g", module.FieldG, "module.FieldG incorrect")
}

func Test_should_append_properties_in_desired_order_when_using_append_props(t *testing.T) {
	module, properties := createTestModuleAndFeatures()
	properties.features["feature_a"] = true
	properties.features["feature_b"] = true
	properties.features["feature_c"] = false
	properties.features["feature_d"] = false

	if err := module.AppendProps([]interface{}{&module}, &properties); err != nil {
		panic(err)
	}
	assert.Equalf(t, "aProps_a", module.FieldA, "module.FieldA incorrect")
	assert.Equalf(t, "bProps_b", module.FieldB, "module.FieldB incorrect")
	assert.Equalf(t, "cProps_c", module.FieldC, "module.FieldC incorrect")
	assert.Equalf(t, "d", module.FieldD, "module.FieldD can't be changed") // No feature has this property
	assert.Equalf(t, "e", module.FieldE, "module.FieldE incorrect")
	assert.Equalf(t, "f", module.FieldF, "module.FieldF incorrect")
	assert.Equalf(t, "gProps_g", module.FieldG, "module.FieldG incorrect")

	properties.features["feature_a"] = false
	properties.features["feature_b"] = false
	properties.features["feature_c"] = false
	properties.features["feature_d"] = true

	if err := module.AppendProps([]interface{}{&module}, &properties); err != nil {
		panic(err)
	}
	assert.Equalf(t, "aProps_a+D_a", module.FieldA, "module.FieldA incorrect")
	assert.Equalf(t, "bProps_b+D_b", module.FieldB, "module.FieldB incorrect")
	assert.Equalf(t, "cProps_c+D_c", module.FieldC, "module.FieldC incorrect")
	assert.Equalf(t, "d", module.FieldD, "module.FieldD can't be changed") // No feature has this property
	assert.Equalf(t, "e+D_e", module.FieldE, "module.FieldE incorrect")
	assert.Equalf(t, "f+D_f", module.FieldF, "module.FieldF incorrect")
	assert.Equalf(t, "gProps_g+D_g", module.FieldG, "module.FieldG incorrect")

	properties.features["feature_a"] = false
	properties.features["feature_b"] = true
	properties.features["feature_c"] = false
	properties.features["feature_d"] = true

	if err := module.AppendProps([]interface{}{&module}, &properties); err != nil {
		panic(err)
	}
	assert.Equalf(t, "aProps_a+D_a+D_a", module.FieldA, "module.FieldA incorrect")
	assert.Equalf(t, "bProps_b+D_bProps_b+D_b", module.FieldB, "module.FieldB incorrect")
	assert.Equalf(t, "cProps_c+D_c+D_c", module.FieldC, "module.FieldC incorrect")
	assert.Equalf(t, "d", module.FieldD, "module.FieldD can't be changed") // No feature has this property
	assert.Equalf(t, "e+D_e+D_e", module.FieldE, "module.FieldE incorrect")
	assert.Equalf(t, "f+D_f+D_f", module.FieldF, "module.FieldF incorrect")
	assert.Equalf(t, "gProps_g+D_g+D_g", module.FieldG, "module.FieldG incorrect")
}

//  It is important that names start from uppercase, otherwise they aren't exported (when nested)
type TestSourceProps struct {
	A string
}

//  It is important that names start from uppercase, otherwise they aren't exported (when nested)
type TestInstallProps struct {
	B string
}

func Test_should_append_properties_when_using_nested_destinations(t *testing.T) {
	type testSource struct {
		Properties struct { // name of nested struct needs to start from capital letter to be exported
			TestSourceProps
		}
	}
	type testInstall struct {
		Properties struct { // name of nested struct needs to start from capital letter to be exported
			TestInstallProps
		}
	}
	type testModule struct {
		testSource
		testInstall
		Properties struct {
			Features // containing COPY of ALL features filled in using reflection (normally done by blueprint)
		}
	}

	// Our module with composited properties
	module := testModule{}
	module.testSource.Properties.A = "mod_a"
	module.testInstall.Properties.B = "mod_b"

	// We need to init all available features (important)
	featureNames := []string{"my_feature_a", "my_feature_b"}
	properties := enabledFeatures(featureNames...)

	module.Properties.Init(&properties, TestSourceProps{}, TestInstallProps{})

	////////////////////////////////////////////////////////////////////////////////////////////////
	// Injecting data to features. This is only for test purpose. Normally this step would be skipped
	// by blueprint and data will be injected from .bp directly to "struct" created by reflection
	module.Properties.injectData("My_feature_a", "A", "+value_a")
	module.Properties.injectData("My_feature_b", "B", "+value_b")
	////////////////////////////////////////////////////////////////////////////////////////////////

	dst := []interface{}{&module.testSource.Properties.TestSourceProps,
		&module.testInstall.Properties.TestInstallProps}

	if err := module.Properties.AppendProps(dst, &properties); err != nil {
		panic(err)
	}

	assert.Equalf(t, "mod_a+value_a", module.testSource.Properties.A, "module.testSource.Properties.A incorrect")
	assert.Equalf(t, "mod_b+value_b", module.testInstall.Properties.B, "module.testInstall.Properties.B incorrect")
}

func Test_should_append_props_when_using_nested_structs(t *testing.T) {
	type TestModuleCommonProps struct {
		TestSourceProps
		TestInstallProps
		Nested struct {
			Foo string
			Bar *bool
		}
	}
	type TestModuleCommon struct {
		Properties struct {
			TestModuleCommonProps
		}
	}
	type TestDerivedModuleProps struct {
		DerivedPropA string
	}
	type testDerivedModule struct {
		TestModuleCommon
		Properties struct {
			TestDerivedModuleProps
			Features // Init'd with TestDerivedModuleProps and TestModuleCommonProps
		}
	}

	// // Our module with composited properties
	module := testDerivedModule{}
	module.TestModuleCommon.Properties.A = "mod_a"
	module.TestModuleCommon.Properties.B = "mod_b"
	module.TestModuleCommon.Properties.Nested.Foo = "mod_foo"
	magicBool := false
	module.TestModuleCommon.Properties.Nested.Bar = &magicBool
	module.TestModuleCommon.Properties.TestSourceProps.A = "mod_TestSourceProps.A"
	module.TestModuleCommon.Properties.TestInstallProps.B = "mod_TestInstallProps.B"

	// We need to init all available features (important)
	featureNames := []string{"my_feature"}
	properties := enabledFeatures(featureNames...)
	module.Properties.Init(&properties, TestDerivedModuleProps{}, TestModuleCommonProps{})

	// This is how 'my_feature' struct will look like
	// My_feature: struct
	// {
	//     DerivedPropA: string
	//     TestSourceProps: core.TestSourceProps
	//     {
	//         A: string
	//     }
	//     TestInstallProps: core.TestInstallProps
	//     {
	//         B: string
	//     }
	//     Nested: struct
	//     {
	//         Foo: string
	//         Bar: *bool
	//     }
	// }
	// If you want to print above uncomment below
	printDebug(reflect.ValueOf(module.Properties.BlueprintEmbed).Elem())

	////////////////////////////////////////////////////////////////////////////////////////////////
	// Injecting data to features. This is only for test purpose. Normally this step would be skipped
	// by blueprint and data will be injected from .bp directly to "struct" created by reflection
	module.Properties.injectData("My_feature", "DerivedPropA", "+feature.DerivedPropA")
	module.Properties.injectData("My_feature", "TestSourceProps.A", "+feature.A")
	module.Properties.injectData("My_feature", "TestInstallProps.B", "+feature.B")
	module.Properties.injectData("My_feature", "Nested.Foo", "+feature.Foo")
	magicBool2 := true
	module.Properties.injectData("My_feature", "Nested.Bar", &magicBool2)
	////////////////////////////////////////////////////////////////////////////////////////////////

	dst := []interface{}{&module.Properties.TestDerivedModuleProps,
		&module.TestModuleCommon.Properties.TestModuleCommonProps}

	if err := module.Properties.AppendProps(dst, &properties); err != nil {
		panic(err)
	}

	assert.Equalf(t, "mod_TestSourceProps.A+feature.A", module.TestModuleCommon.Properties.A,
		"module.TestModuleCommon.Properties.A incorrect")
	assert.Equalf(t, "mod_TestInstallProps.B+feature.B", module.TestModuleCommon.Properties.B,
		"module.TestModuleCommon.Properties.B incorrect")
	assert.Equalf(t, true, *module.TestModuleCommon.Properties.Nested.Bar,
		" module.TestModuleCommon.Properties.Nested.Bar incorrect")
	assert.Equalf(t, "mod_foo+feature.Foo", module.TestModuleCommon.Properties.Nested.Foo,
		"module.TestModuleCommon.Properties.Nested.Foo incorrect")
	assert.Equalf(t, "mod_TestSourceProps.A+feature.A", module.TestModuleCommon.Properties.TestSourceProps.A,
		"module.TestModuleCommon.Properties.TestSourceProps.A incorrect")
	assert.Equalf(t, "mod_TestInstallProps.B+feature.B", module.TestModuleCommon.Properties.TestInstallProps.B,
		"module.TestModuleCommon.Properties.TestInstallProps.B incorrect")
}

func Test_should_not_squash_when_one_structs_passed(t *testing.T) {
	squashed := coalesceTypes(typesOf(testPropsGroupA{})...)
	assert.Equalf(t, squashed, reflect.TypeOf(testPropsGroupA{}), "Types should be the same")
}

func Test_should_composite_new_type(t *testing.T) {
	module := testProps{
		FieldA: "a",
		FieldB: "b",
		FieldC: "c",
		FieldD: "d",
		FieldE: "e",
		FieldF: "f",
		FieldG: "g",
	}

	// We need to init all available features (important)
	properties := enabledFeatures("feature_compose")
	module.Init(&properties,
		testPropsGroupA{},
		testPropsGroupB{},
	)

	// Uncomment if you want to "view for development purposes"
	//printDebug(reflect.ValueOf(module.BlueprintEmbed).Elem())
	// Above function will print:
	// {
	//   Feature_compose: struct { FieldA string; FieldC string; FieldF string; FieldB string }
	//   {
	//       FieldA: string
	//       FieldC: string
	//       FieldF: string
	//       FieldB: string
	//   }
	// }

	// Below code shouldn't fail
	propsInFeature := reflect.ValueOf(module.BlueprintEmbed).Elem().FieldByName("Feature_compose").Interface().(singleFeature)
	feature := reflect.ValueOf(propsInFeature.BlueprintEmbed).Elem()
	feature.FieldByName("FieldA").SetString("+value_a")
	feature.FieldByName("FieldB").SetString("+value_b")
}
