package plugin

import (
	"reflect"
	"testing"

	mparser "github.com/ARM-software/bob-build/gazelle/mconfig/parser"
	"github.com/stretchr/testify/assert"
)

func TestBobConfigSpoof(t *testing.T) {

	var configs map[string]*mparser.ConfigData = make(map[string]*mparser.ConfigData)

	configs["feature_x"] = &mparser.ConfigData{Type: "config", Datatype: "bool"}
	configs["feature_y"] = &mparser.ConfigData{Type: "config", Datatype: "string"}

	bobConfig := createBobConfigSpoof(&configs)

	assert.Equal(t, len(bobConfig.Properties.FeatureList), 2, "Wrong features inside config")
	assert.Contains(t, bobConfig.Properties.FeatureList, "feature_x")
	assert.Contains(t, bobConfig.Properties.FeatureList, "feature_y")

	if v, ok := bobConfig.Properties.Properties["feature_x"].(*mparser.ConfigData); ok {
		assert.Equal(t, "bool", v.Datatype)
		assert.Equal(t, "config", v.Type)
	} else {
		t.Errorf("configData.Datatype of 'feature_x' has wrong type: %s", reflect.TypeOf(v))
	}

	if v, ok := bobConfig.Properties.Properties["feature_y"].(*mparser.ConfigData); ok {
		assert.Equal(t, "string", v.Datatype)
		assert.Equal(t, "config", v.Type)
	} else {
		t.Errorf("configData.Datatype of 'feature_y' has wrong type: %s", reflect.TypeOf(v))
	}
}
