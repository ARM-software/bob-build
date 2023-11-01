package plugin

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ARM-software/bob-build/gazelle/common"
	"github.com/ARM-software/bob-build/gazelle/registry"
	"github.com/ARM-software/bob-build/gazelle/types"
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
	idx          uint32
	registry     *registry.Registry
}

func (m *Module) GetName() string {
	return m.name
}

func (m *Module) GetRelativePath() string {
	return m.relativePath
}

func (m *Module) GetLabel() label.Label {
	return m.bazelLabel
}

func (m *Module) SetRegistry(r *registry.Registry) {
	m.registry = r
}

func (m *Module) addDefaultAttribute(attribute string, a Attribute) {
	m.defaults[attribute] = a
}

func (m *Module) AddFeatureAttribute(feature string, attribute string, a Attribute) {

	if feature == common.ConditionDefault {
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
	// Repo is set to "" as we do not expect to support
	// external workspace dependencies.
	m.bazelLabel = label.Label{Repo: "", Pkg: strings.TrimLeft(relPath, "."), Name: moduleName}
	m.features = make(map[string]AttributesMap)
	m.defaults = make(AttributesMap)

	return m
}

func (m *Module) GenerateRule() (r *rule.Rule, err error) {

	switch m.moduleType {
	case ModuleFilegroup:
		// build Bazel `filegroup` from `bob_filegroup`
		r = m.buildFilegroup()
	case ModuleGlob:
		// build Bazel `filegroup` from `bob_glob`
		r = m.buildGlobFilegroup()
	case ModuleLibrary:
		r = m.buildLibrary()
	default:
		err = fmt.Errorf("Unsupported module '%s'", m.moduleType)
		return nil, err
	}

	return r, nil
}

func resolveLabels(l []string, m *Module) []string {
	var resolved []string
	registry := m.registry
	for _, v := range l {
		if registry.NameExists(v) {
			if dep, ok := registry.RetrieveByName(v); ok {
				depLabel := dep.GetLabel()
				relativeDepLabel := depLabel.Rel(m.bazelLabel.Repo, m.bazelLabel.Pkg)
				// makes path relative
				resolved = append(resolved, relativeDepLabel.String())
			}
		} else {
			resolved = append(resolved, v)
		}
	}
	return resolved
}

func buildListExpressionFromAttribute(m *Module, attr string) bzl.Expr {
	list := make([]bzl.Expr, 0, 1+len(m.features))

	if d, ok := m.defaults[attr]; ok {
		if srcs, ok := d.([]string); ok {
			list = append(list, types.MakeStringListWithGlob(resolveLabels(srcs, m)).BzlExpr())
		}
	}

	features := make([]string, len(m.features))
	keys := reflect.ValueOf(m.features).MapKeys()

	for i, k := range keys {
		features[i] = k.String()
	}

	sort.Strings(features)

	for _, name := range features {
		attribute := m.features[name]
		data := make(types.SelectStringListWithGlob)
		if srcs, ok := attribute[attr].([]string); ok {
			data[common.GetFeatureCondition(name)] = types.MakeStringListWithGlob(resolveLabels(srcs, m))
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

	return expr
}

func buildBooleanExpressionFromAttribute(m *Module, attr string) (expr *bzl.LiteralExpr, err bool) {
	if d, ok := m.defaults[attr]; ok {
		if val, ok := d.(*bool); ok {
			tok := "False"
			if *val {
				tok = "True"
			}
			return &bzl.LiteralExpr{Token: tok}, true
		}
	}

	return nil, false
}

func (m *Module) buildLibrary() *rule.Rule {
	// TODO: Currently only correctly translates StrictLibraryProps right now.
	r := rule.NewRule(m.moduleType.String(), m.name)
	r.SetKind(m.moduleType.String())

	// These set of attributes are 1 to 1 string lists that have additive conditionals only.
	if attr := buildListExpressionFromAttribute(m, "Srcs"); attr != nil {
		r.SetAttr("srcs", &types.SrcsAttribute{Expr: attr})
	}
	if attr := buildListExpressionFromAttribute(m, "Hdrs"); attr != nil {
		r.SetAttr("hdrs", &types.SrcsAttribute{Expr: attr})
	}
	if attr := buildListExpressionFromAttribute(m, "Local_defines"); attr != nil {
		r.SetAttr("local_defines", attr)
	}
	if attr := buildListExpressionFromAttribute(m, "Defines"); attr != nil {
		r.SetAttr("defines", attr)
	}
	if attr := buildListExpressionFromAttribute(m, "Copts"); attr != nil {
		r.SetAttr("copts", attr)
	}
	if attr := buildListExpressionFromAttribute(m, "Deps"); attr != nil {
		r.SetAttr("deps", attr)
	}

	if expr, ok := buildBooleanExpressionFromAttribute(m, "Alwayslink"); ok {
		r.SetAttr("alwayslink", expr)
	}
	if expr, ok := buildBooleanExpressionFromAttribute(m, "Linkstatic"); ok {
		r.SetAttr("linkstatic", expr)
	}

	return r
}

func (m *Module) buildFilegroup() *rule.Rule {

	r := rule.NewRule(m.moduleType.String(), m.name)
	r.SetKind(m.moduleType.String())

	list := make([]bzl.Expr, 0, 1+len(m.features))

	if d, ok := m.defaults["Srcs"]; ok {
		if srcs, ok := d.([]string); ok {
			list = append(list, types.MakeStringListWithGlob(srcs).BzlExpr())
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
		data := make(types.SelectStringListWithGlob)
		if srcs, ok := attr["Srcs"].([]string); ok {
			data[common.GetFeatureCondition(name)] = types.MakeStringListWithGlob(srcs)
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

	r.SetAttr("srcs", &types.SrcsAttribute{Expr: expr})

	return r
}

func (m *Module) buildGlobFilegroup() *rule.Rule {

	r := rule.NewRule(m.moduleType.String(), m.name)
	r.SetKind(m.moduleType.String())

	g := &types.GlobValue{}

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

	r.SetAttr("srcs", &types.SrcsAttribute{Expr: g.BzlExpr()})

	return r
}

func (m *Module) SetIndex(i uint32) {
	m.idx = i
}

func (m *Module) GetIndex() uint32 {
	return m.idx
}
