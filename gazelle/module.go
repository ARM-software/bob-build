package plugin

import (
	"github.com/bazelbuild/bazel-gazelle/label"
)

type AttributesMap map[string]interface{}
type BobModule struct {
	bobName      string
	relativePath string
	bazelLabel   label.Label
	features     map[string]AttributesMap
}

func (m BobModule) getName() string {
	return m.bobName
}

func (m BobModule) getRelativePath() string {
	return m.relativePath
}

func (m BobModule) getLabel() label.Label {
	return m.bazelLabel
}

func (m *BobModule) addFeatureAttribute(feature string, attribute string, v interface{}) {

	if f, ok := m.features[feature]; ok {
		f[attribute] = v
	} else {
		m.features[feature] = make(AttributesMap)
		m.features[feature][attribute] = v
	}
}

func NewBobModule(name string, repoRoot string, relPath string) *BobModule {

	bobModule := &BobModule{}
	bobModule.bobName = name
	bobModule.relativePath = relPath
	bobModule.bazelLabel = label.Label{Repo: repoRoot, Pkg: relPath, Name: name}
	bobModule.features = make(map[string]AttributesMap)

	return bobModule
}
