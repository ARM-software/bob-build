package plugin

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
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
	case ModuleFilegroup, ModuleGlob:
		return "filegroup"
	}
	return "unknown"
}

func ParseModuleType(str string) ModuleType {
	if t, ok := bobToModuleTypeMap[strings.ToLower(str)]; ok {
		return t
	} else {
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

func (m *Module) generateRule() (r *rule.Rule, err error) {

	switch m.moduleType {
	case ModuleFilegroup:
		// build Bazel `filegroup` from `bob_filegroup`
		r = m.buildFilegroup()
	case ModuleGlob:
		// build Bazel `filegroup` from `bob_glob`
		r = m.buildGlobFilegroup()
	default:
		err = fmt.Errorf("Unsupported module '%s'", m.moduleType)
		return nil, err
	}

	return r, nil
}

func (m *Module) buildFilegroup() *rule.Rule {

	r := rule.NewRule(m.moduleType.String(), m.name)
	r.SetKind(m.moduleType.String())

	list := make([]bzl.Expr, 0, 2)

	if d, ok := m.features[ConditionDefault]; ok {
		if srcs, ok := d["Srcs"].([]string); ok {
			list = append(list, makeStringListWithGlob(srcs).BzlExpr())
		}
	}

	data := make(SelectStringListWithGlobValue)

	for name, attr := range m.features {
		if srcs, ok := attr["Srcs"].([]string); ok && name != ConditionDefault {
			data[getFeatureCondition(name)] = makeStringListWithGlob(srcs)
		}
	}

	if len(data) > 0 {
		list = append(list, data.SelectWithGlob())
	}

	if len(list) == 2 {
		r.SetAttr("srcs", &bzl.BinaryExpr{X: list[0], Y: list[1], Op: "+"})
	} else if len(list) == 1 {
		r.SetAttr("srcs", list[0])
	}

	return r
}

func (m *Module) buildGlobFilegroup() *rule.Rule {

	r := rule.NewRule(m.moduleType.String(), m.name)
	r.SetKind(m.moduleType.String())

	g := &GlobValue{}

	// `bob_glob` is not featurable thus only `ConditionDefault` is present

	if d, ok := m.features[ConditionDefault]; ok {
		if srcs, ok := d["Srcs"].([]string); ok {
			g.Patterns = srcs
		}
		if allowEmpty, ok := d["Allow_empty"].(*bool); ok {
			g.AllowEmpty = allowEmpty
		}
		if exclude, ok := d["Exclude"].([]string); ok {
			g.Excludes = exclude
		}
		if excludeDir, ok := d["Exclude_directories"].(*bool); ok {
			g.ExcludeDirectories = excludeDir
		}
	}

	r.SetAttr("srcs", g.BzlExpr())

	return r
}

// TODO: resolve feature names properly depending on
// the location in `build.bp`
func getFeatureCondition(f string) string {
	if f == ConditionDefault {
		return f
	} else {
		return fmt.Sprint(":", strings.ToLower(f))
	}
}
