package plugin

import (
	"github.com/bazelbuild/bazel-gazelle/label"
)

type AttributesMap map[string]interface{}
type Module struct {
	bobName      string
	bobType      string
	relativePath string
	bazelLabel   label.Label
	features     map[string]AttributesMap
}

func (m Module) getName() string {
	return m.bobName
}

func (m Module) getRelativePath() string {
	return m.relativePath
}

func (m Module) getLabel() label.Label {
	return m.bazelLabel
}

func (m *Module) addFeatureAttribute(feature string, attribute string, v interface{}) {

	if f, ok := m.features[feature]; ok {
		f[attribute] = v
	} else {
		m.features[feature] = make(AttributesMap)
		m.features[feature][attribute] = v
	}
}

func NewModule(moduleName string, moduleType string, relPath string, rootPath string) *Module {

	m := &Module{}
	m.bobName = moduleName
	m.bobType = moduleType
	m.relativePath = relPath
	m.bazelLabel = label.Label{Repo: rootPath, Pkg: relPath, Name: moduleName}
	m.features = make(map[string]AttributesMap)

	return m
}
