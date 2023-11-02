package parser

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBobConfigSpoof(t *testing.T) {

	var configs map[string]*ConfigData = make(map[string]*ConfigData)

	configs["feature_x"] = &ConfigData{Type: "config", Datatype: "bool"}
	configs["feature_y"] = &ConfigData{Type: "config", Datatype: "string"}

	bobConfig := CreateBobConfigSpoof(&configs)

	assert.Equal(t, len(bobConfig.Properties.FeatureList), 2, "Wrong features inside config")
	assert.Contains(t, bobConfig.Properties.FeatureList, "feature_x")
	assert.Contains(t, bobConfig.Properties.FeatureList, "feature_y")

	if v, ok := bobConfig.Properties.Properties["feature_x"].(*ConfigData); ok {
		assert.Equal(t, "bool", v.Datatype)
		assert.Equal(t, "config", v.Type)
	} else {
		t.Errorf("configData.Datatype of 'feature_x' has wrong type: %s", reflect.TypeOf(v))
	}

	if v, ok := bobConfig.Properties.Properties["feature_y"].(*ConfigData); ok {
		assert.Equal(t, "string", v.Datatype)
		assert.Equal(t, "config", v.Type)
	} else {
		t.Errorf("configData.Datatype of 'feature_y' has wrong type: %s", reflect.TypeOf(v))
	}
}
