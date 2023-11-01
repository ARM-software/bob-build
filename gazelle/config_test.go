package plugin

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBobConfigSpoof(t *testing.T) {

	var configs map[string]*configData = make(map[string]*configData)

	configs["feature_x"] = &configData{Type: "config", Datatype: "bool"}
	configs["feature_y"] = &configData{Type: "config", Datatype: "string"}

	bobConfig := createBobConfigSpoof(&configs)

	assert.Equal(t, len(bobConfig.Properties.FeatureList), 2, "Wrong features inside config")
	assert.Contains(t, bobConfig.Properties.FeatureList, "feature_x")
	assert.Contains(t, bobConfig.Properties.FeatureList, "feature_y")

	if v, ok := bobConfig.Properties.Properties["feature_x"].(*configData); ok {
		assert.Equal(t, "bool", v.Datatype)
		assert.Equal(t, "config", v.Type)
	} else {
		t.Errorf("configData.Datatype of 'feature_x' has wrong type: %s", reflect.TypeOf(v))
	}

	if v, ok := bobConfig.Properties.Properties["feature_y"].(*configData); ok {
		assert.Equal(t, "string", v.Datatype)
		assert.Equal(t, "config", v.Type)
	} else {
		t.Errorf("configData.Datatype of 'feature_y' has wrong type: %s", reflect.TypeOf(v))
	}
}
