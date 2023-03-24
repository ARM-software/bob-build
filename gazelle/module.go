package plugin

import (
	"fmt"
	"reflect"
	"sort"
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

type Attribute interface{}
type AttributesMap map[string]Attribute

type Module struct {
	name         string
	bobType      string
	moduleType   ModuleType
	relativePath string
	bazelLabel   label.Label
	features     map[string]AttributesMap
	defaults     AttributesMap
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

func (m *Module) addDefaultAttribute(attribute string, a Attribute) {
	m.defaults[attribute] = a
}

func (m *Module) addFeatureAttribute(feature string, attribute string, a Attribute) {

	if feature == ConditionDefault {
		m.addDefaultAttribute(attribute, a)
	} else if f, ok := m.features[feature]; ok {
		f[attribute] = a
	} else {
		m.features[feature] = make(AttributesMap)
		m.features[feature][attribute] = a
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
	m.defaults = make(AttributesMap)

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

	list := make([]bzl.Expr, 0, 1+len(m.features))

	if d, ok := m.defaults["Srcs"]; ok {
		if srcs, ok := d.([]string); ok {
			list = append(list, makeStringListWithGlob(srcs).BzlExpr())
		}
	}

	// features have to be sorted to preserve generation order
	// TODO: improve sorting
	features := make([]string, len(m.features))
	keys := reflect.ValueOf(m.features).MapKeys()

	for i, k := range keys {
		features[i] = k.String()
	}

	sort.Strings(features)

	for _, name := range features {
		attr := m.features[name]
		data := make(SelectStringListWithGlob)
		if srcs, ok := attr["Srcs"].([]string); ok {
			data[getFeatureCondition(name)] = makeStringListWithGlob(srcs)
			list = append(list, data.BzlExpr())
		}
	}

	var expr bzl.Expr

	if len(list) > 0 {
		expr = list[0]

		for _, l := range list[1:] {
			expr = &bzl.BinaryExpr{X: expr, Y: l, Op: "+"}
		}
	}

	r.SetAttr("srcs", expr)

	return r
}

func (m *Module) buildGlobFilegroup() *rule.Rule {

	r := rule.NewRule(m.moduleType.String(), m.name)
	r.SetKind(m.moduleType.String())

	g := &GlobValue{}

	// `bob_glob` is not featurable
	// thus only `m.defaults` data is used

	if srcs, ok := m.defaults["Srcs"]; ok {
		g.Patterns = srcs.([]string)
	}
	if allowEmpty, ok := m.defaults["Allow_empty"]; ok {
		g.AllowEmpty = allowEmpty.(*bool)
	}
	if exclude, ok := m.defaults["Exclude"]; ok {
		g.Excludes = exclude.([]string)
	}
	if excludeDir, ok := m.defaults["Exclude_directories"]; ok {
		g.ExcludeDirectories = excludeDir.(*bool)
	}

	r.SetAttr("srcs", g.BzlExpr())

	return r
}
