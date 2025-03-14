package core

import (
	"reflect"
	"strings"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/internal/utils"
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
//
//	type BlueprintEmbedType struct {
//	        Debug PropsType
//	        Enable_something PropsType
//	        Some_other_feature PropsType
//	}
//
// Name of each property in this struct is custom feature name.
// Blueprint will inflate this structure with data read from .bp files.
// Only exported properties can be set so property name MUST start from capital letter.
func (f *Features) Init(properties *config.Properties, list ...interface{}) {
	if len(list) == 0 {
		utils.Die("List can't be empty")
	}

	propsType := coalesceTypes(typesOf(list...)...)
	fields := make([]reflect.StructField, len(properties.FeatureList))

	for i, featureName := range properties.FeatureList {
		fields[i] = reflect.StructField{
			Name: featurePropertyName(featureName),
			Type: reflect.PtrTo(propsType),
		}
	}

	bpFeatureStruct := reflect.StructOf(fields)
	instancePtr := reflect.New(bpFeatureStruct)
	f.BlueprintEmbed = instancePtr.Interface()

}

// Set internal `BlueprintEmbed` field to nil.
//
// Use it carefully as features won't be available anymore.
func (f *Features) DeInit() {
	f.BlueprintEmbed = nil
}

// coalesceTypes will squash multiple types to new type. This has different result
// than Go composition of structs.
//
// Example (go composition):
//
//	type compositeStruct struct {
//	    testPropsGroupA
//	    testPropsGroupB
//	}
//
// Debug print:
//
//	{
//	    testPropsGroupA: core.testPropsGroupA
//	    {
//	      Field_a: string
//	      Field_c: string
//	      Field_f: string
//	    }
//	    testPropsGroupB: core.testPropsGroupB
//	    {
//	      Field_b: string
//	    }
//	}
//
//	Example for: coalesceTypes([]reflect.Type{
//	   testPropsGroupA{},
//	   testPropsGroupB{},
//	})
//
// Debug print:
//
//	{
//	    Field_a: string
//	    Field_c: string
//	    Field_f: string
//	    Field_b: string
//	}
func coalesceTypes(list ...reflect.Type) reflect.Type {
	if len(list) == 0 {
		utils.Die("List can't be empty")
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
				utils.Die("Name collision: '%v'\n", fieldName)
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
func (f *Features) AppendProps(dst []interface{}, properties *config.Properties) error {
	// featuresData is struct created in Features.Init function
	featuresData := reflect.ValueOf(f.BlueprintEmbed).Elem()

	for _, featureKey := range properties.FeatureList {
		if properties.Features[featureKey] { // Check the feature is enabled
			// Features are matched like "Feature_name" - feature structure
			featureFieldName := featurePropertyName(featureKey)
			featureStruct := featuresData.FieldByName(featureFieldName)
			if !featureStruct.IsValid() {
				utils.Die("Field returned for property %s isn't valid\n", featureFieldName)
			}

			// If featureProps is nil then we've determined that we can skip this,
			// so avoid calling AppendMatchingProperties
			if !featureStruct.IsNil() {
				err := AppendMatchingProperties(dst, featureStruct.Interface())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
