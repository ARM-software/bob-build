package plugin

import (
	"fmt"
	"reflect"
	"testing"

	bob "github.com/ARM-software/bob-build/core"
	"github.com/ARM-software/bob-build/gazelle/common"
	mod "github.com/ARM-software/bob-build/gazelle/module"
	"github.com/google/blueprint"
	"github.com/stretchr/testify/assert"
)

type SourceProps struct {
	Srcs      []string
	HidenSrcs string `blueprint:"mutated"`
}

type testModule struct {
	blueprint.SimpleName
	Properties struct {
		SourceProps
		bob.Features
	}
}

func (m testModule) Name() string {
	return m.SimpleName.Properties.Name
}

func (m testModule) GetProperties() interface{} {
	return m.Properties
}

func (m *testModule) GenerateBuildActions(blueprint.ModuleContext) {
	// pass
}

func TestParseBpModule(t *testing.T) {

	features := make(map[string]mod.AttributesMap)

	bobConfig := &bob.BobConfig{}

	bobConfig.Properties.FeatureList = make([]string, 2)
	bobConfig.Properties.FeatureList[0] = "FEATURE_X"
	bobConfig.Properties.FeatureList[1] = "FEATURE_Y"

	bobConfig.Properties.Features = make(map[string]bool)
	bobConfig.Properties.Features["FEATURE_X"] = true
	bobConfig.Properties.Features["FEATURE_Y"] = true

	bobConfig.Properties.Properties = make(map[string]interface{})
	bobConfig.Properties.Properties["FEATURE_X"] = configData{}
	bobConfig.Properties.Properties["FEATURE_Y"] = configData{}

	injectFeature := func(features *bob.Features, featureName string, v interface{}) {
		t.Helper()
		var isSet bool = false

		fType := reflect.ValueOf(features.BlueprintEmbed).Elem().Type()
		fValue := reflect.ValueOf(features.BlueprintEmbed).Elem()

		for i := 0; i < fType.NumField(); i++ {
			name := fType.Field(i).Name

			if fType.Field(i).IsExported() && name == featureName {
				field := fValue.FieldByName(name).FieldByName("BlueprintEmbed")

				if reflect.TypeOf(field.Interface()) == reflect.TypeOf(v) && field.CanSet() {
					field.Set(reflect.ValueOf(v))
					isSet = true
					break
				}
			}
		}

		if !isSet {
			t.Errorf("no feature '%s' or couldn't set", featureName)
		}
	}

	bpModule := &testModule{
		Properties: struct {
			SourceProps
			bob.Features
		}{
			SourceProps: SourceProps{
				Srcs: []string{"main.c"},
			},
		},
	}

	handler := func(feature string, attribute string, v interface{}) {
		if f, ok := features[feature]; ok {
			f[attribute] = v
		} else {
			features[feature] = make(mod.AttributesMap)
			features[feature][attribute] = v
		}
	}

	bpModule.Properties.Features.Init(&bobConfig.Properties, SourceProps{})

	injectFeature(&bpModule.Properties.Features, "Feature_y", &SourceProps{Srcs: []string{"libA.c"}})

	parseBpModule(bpModule, handler)

	assert.Equal(t, 2, len(features), "Wrong features count")
	assert.Contains(t, features, common.ConditionDefault, fmt.Sprintf("No '%s' found", common.ConditionDefault))
	assert.Contains(t, features, "Feature_y", "No 'Feature_y' found")

	for _, f := range features {
		assert.Contains(t, f, "Srcs")
		if v, ok := f["Srcs"].([]string); ok {
			assert.Equal(t, 1, len(v))
		} else {
			t.Errorf("Wrong type of 'Srcs' (%s)", reflect.TypeOf(f["Srcs"]))
		}
	}

	assert.Contains(t, features[common.ConditionDefault]["Srcs"], "main.c")
	assert.Contains(t, features["Feature_y"]["Srcs"], "libA.c")
}
