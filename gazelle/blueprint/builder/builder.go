package builder

import (
	"strings"

	"github.com/ARM-software/bob-build/gazelle/blueprint/builder/attr"
	"github.com/ARM-software/bob-build/gazelle/blueprint/builder/bp2bzl"
	"github.com/ARM-software/bob-build/gazelle/info"
	lb "github.com/ARM-software/bob-build/gazelle/logic/builder"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

const ModuleTypeAny = "*"  // Matches all module types
const UnknownRuleName = "" // Placeholder for brand new rule.

type Builder struct {
	m *mapper.Mapper

	lb *lb.Builder

	// Maps from to attr names per module
	attrNameMapping map[string]map[string]string

	// Maps from to attr names per module
	attrTypeMapping map[string]map[string]attr.AttrType
}

func NewBuilder(m *mapper.Mapper, lb *lb.Builder) *Builder {
	return &Builder{
		m:  m,
		lb: lb,

		attrNameMapping: map[string]map[string]string{},
		attrTypeMapping: map[string]map[string]attr.AttrType{},
	}
}

func (b *Builder) ConfigureDefault() {
	b.ConfigureAttribute(ModuleTypeAny, attr.AttrTypeImmutable, "name", "")
	b.ConfigureAttribute(ModuleTypeAny, attr.AttrTypeAdditive, "hdrs", "")
	b.ConfigureAttribute(ModuleTypeAny, attr.AttrTypeAdditive, "srcs", "")
	b.ConfigureAttribute(ModuleTypeAny, attr.AttrTypeAdditive, "deps", "")
	b.ConfigureAttribute("bob_library", attr.AttrTypeAdditive, "defines", "")
	b.ConfigureAttribute("bob_library", attr.AttrTypeAdditive, "local_defines", "")
	b.ConfigureAttribute("bob_library", attr.AttrTypeSelective, "alwayslink", "")
	b.ConfigureAttribute("bob_library", attr.AttrTypeSelective, "linkstatic", "")
}

func (b *Builder) constructAttributes(mod *parser.Module) (attrs []attr.Attribute) {

	m := map[string]attr.Attribute{}
	for _, prop := range mod.Map.Properties {

		// Check for Mconfig feature by querying the label mapper
		label := b.m.FromValue(strings.ToUpper(prop.Name))

		// fmt.Printf("l:%v %v\n", label, &logic.Identifier{strings.ToUpper(prop.Name)})
		if label != nil {
			for _, featured := range prop.Value.(*parser.Map).Properties {
				if m[featured.Name] == nil {
					m[featured.Name] = attr.NewAttribute(
						featured.Name,
						b.getAttrMapping(mod.Type, featured.Name),
						b.getAttrType(mod.Type, featured.Name),
						b.m,
						b.lb,
					)
				}

				m[featured.Name].SetFeatureValue(strings.ToUpper(prop.Name), featured)
			}
		} else {
			m[prop.Name] = attr.NewAttribute(
				prop.Name,
				b.getAttrMapping(mod.Type, prop.Name),
				b.getAttrType(mod.Type, prop.Name),
				b.m,
				b.lb,
			)
			m[prop.Name].SetValue(prop)
		}
	}

	for _, v := range m {
		if _, ignore := v.(*attr.Ignored); ignore {
			continue
		}
		attrs = append(attrs, v)
	}

	return
}

func (b *Builder) getAttrMapping(bobType string, from string) string {
	anyModule := b.attrNameMapping[ModuleTypeAny]
	moduleScoped := b.attrNameMapping[bobType]

	if moduleScoped != nil {
		if to, ok := moduleScoped[from]; ok {
			return to
		}
	}

	if anyModule != nil {
		if to, ok := anyModule[from]; ok {
			return to
		}
	}

	return from
}

func (b *Builder) setAttrName(bobType string, from string, to string) {
	if b.attrNameMapping[bobType] == nil {
		b.attrNameMapping[bobType] = map[string]string{}
	}
	b.attrNameMapping[bobType][from] = to
}

func (b *Builder) getAttrType(bobType string, from string) attr.AttrType {
	anyModule := b.attrTypeMapping[ModuleTypeAny]
	moduleScoped := b.attrTypeMapping[bobType]

	if moduleScoped != nil {
		if t, ok := moduleScoped[from]; ok {
			return t
		}
	}

	if anyModule != nil {
		if t, ok := anyModule[from]; ok {
			return t
		}
	}

	return attr.AttrTypeIgnored
}

func (b *Builder) setAttrType(bobType string, from string, t attr.AttrType) {
	if b.attrTypeMapping[bobType] == nil {
		b.attrTypeMapping[bobType] = map[string]attr.AttrType{}
	}
	b.attrTypeMapping[bobType][from] = t
}

func (b *Builder) createRule(mod *parser.Module) (*rule.Rule, error) {

	bob2Bazel := map[string]string{
		"bob_binary":         "cc_binary",
		"bob_static_library": "cc_library",
		"bob_shared_library": "cc_library",
		"bob_library":        "cc_library",
		"bob_executable":     "cc_binary",
		"bob_test":           "cc_test",
		"bob_genrule":        "genrule",
		"bob_filegroup":      "filegroup",
		"bob_glob":           "filegroup",
	}

	r := rule.NewRule("", UnknownRuleName)
	if kind, ok := bob2Bazel[mod.Type]; ok {
		r.SetKind(kind)
	} else {
		return nil, nil
	}

	globArgs := []bzl.Expr{}

	switch mod.Type {
	case "bob_glob":
		valueTransformer := &bp2bzl.Transformer{
			Mapper:         nil,
			ResolveTargets: false,
			ConvertGlobs:   false,
		}
		if prop, ok := mod.Map.GetProperty("allow_empty"); ok {
			globArgs = append(globArgs, &bzl.AssignExpr{
				LHS: &bzl.LiteralExpr{Token: "allow_empty"},
				Op:  "=",
				RHS: valueTransformer.Transform(prop.Value),
			})
		}

		if prop, ok := mod.Map.GetProperty("exclude"); ok {
			globArgs = append(globArgs, &bzl.AssignExpr{
				LHS: &bzl.LiteralExpr{Token: "exclude"},
				Op:  "=",
				RHS: valueTransformer.Transform(prop.Value),
			})
		}

		if prop, ok := mod.Map.GetProperty("exclude_directories"); ok {
			globArgs = append(globArgs, &bzl.AssignExpr{
				LHS: &bzl.LiteralExpr{Token: "exclude_directories"},
				Op:  "=",
				RHS: valueTransformer.Transform(prop.Value),
			})
		}

	}

	for _, attr := range b.constructAttributes(mod) {
		if len(globArgs) > 0 {
			attr.SetGlobArgs(globArgs)
		}

		// deps := b.logic.GenerateConfigSetting(attr)

		// fmt.Printf("attr:%#v\n", attr)
		r.SetAttr(attr.ToName(), attr)

	}

	return r, nil
}

func (b *Builder) Build(args language.GenerateArgs, file interface{}) (result language.GenerateResult) {
	rules := []*rule.Rule{}

	for _, def := range file.(*parser.File).Defs {
		switch def := def.(type) {
		case *parser.Module:
			if r, err := b.createRule(def); err == nil && r != nil {
				rules = append(rules, r)
			}
		}
	}

	for _, r := range rules {
		if r.IsEmpty(info.Kinds[r.Kind()]) {
			result.Empty = append(result.Empty, r)

		} else {
			result.Gen = append(result.Gen, r)
			result.Imports = append(result.Imports, r.PrivateAttr(""))
		}
	}
	return
}

func (b *Builder) ConfigureAttribute(bobType string, attrType attr.AttrType, from, to string) {
	if to == "" {
		to = from
	}

	b.setAttrName(bobType, from, to)
	b.setAttrType(bobType, from, attrType)
}
