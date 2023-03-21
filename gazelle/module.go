package plugin

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
)

type ModuleType int8

const (
	ModuleUndefined ModuleType = iota
	ModuleBinary
	ModuleLibrary
	ModuleFilegroup
	ModuleGlob
)

var (
	bobToModuleTypeMap = map[string]ModuleType{
		"bob_filegroup":       ModuleFilegroup,
		"bob_glob":            ModuleGlob,
		"bob_binary":          ModuleBinary,
		"bob_static_library":  ModuleLibrary,
		"bob_dynamic_library": ModuleLibrary,
		"bob_library":         ModuleLibrary,
	}
)

func (t ModuleType) String() string {
	switch t {
	case ModuleBinary:
		return "cc_binary"
	case ModuleLibrary:
		return "cc_library"
	case ModuleFilegroup:
		return "filegroup"
	}
	return "unknown"
}

func ParseModuleType(str string) ModuleType {
	if t, ok := bobToModuleTypeMap[strings.ToLower(str)]; ok {
		return t
	} else {
		log.Printf("Undefined module type: %s\n", str)
		return ModuleUndefined
	}
}

type AttributesMap map[string]interface{}

type Module struct {
	name         string
	bobType      string
	moduleType   ModuleType
	relativePath string
	bazelLabel   label.Label
	features     map[string]AttributesMap
}

func (m Module) getName() string {
	return m.name
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
	m.name = moduleName
	m.bobType = moduleType
	m.moduleType = ParseModuleType(moduleType)
	m.relativePath = relPath
	m.bazelLabel = label.Label{Repo: rootPath, Pkg: relPath, Name: moduleName}
	m.features = make(map[string]AttributesMap)

	return m
}
