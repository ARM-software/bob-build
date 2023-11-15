// Given the following module:
//```
//bob_module {
//	attr: <base_value>,
//	<feature0>: {
//		attr: <feature0_value>,
//	},
//	<feature1>: {
//		attr: <feature1_value>,
//	},
//	...
//	<featureN>: {
//		attr: <featureN_value>,
//	}
//}
//```
// This would result in the following truth table:
//```
// f0 f1 fN
// 0  0  0 --> default
// 1  0  0 --> feature0_value
// X  1  0 --> feature1_value
// X  X  1 --> featureN_value
//```
// We can translate this logic to a select statement:
//```
// select(
// 	"!f0 && !f1 && !fN" : default
// 	" f0 && !f1 && !fN" : feature0_value
// 	"        f1 && !fN" : feature1_value
// 	"               fN" : featureN_value
// )
//```

package attr

import (
	"fmt"

	"github.com/ARM-software/bob-build/gazelle/blueprint/builder/bp2bzl"
	"github.com/ARM-software/bob-build/gazelle/logic"
	lb "github.com/ARM-software/bob-build/gazelle/logic/builder"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

type Selective struct {
	from, to string // Source and destination attribute names, allows mapping from Bob attr name to Bazel attr name
	globArgs []bzl.Expr
	base     *parser.Property

	rel string
	m   *mapper.Mapper
	lb  *lb.Builder

	featureProps map[string]*parser.Property
	featureNames []string
}

var _ Attribute = (*Selective)(nil) // impl check

func (a *Selective) createKeyValueEntry(expr logic.Expr, p *parser.Property) *bzl.KeyValueExpr {
	return nil
}

func (a *Selective) createBaseCase() *bzl.KeyValueExpr {
	t := &bp2bzl.Transformer{
		ResolveTargets: true,
		GlobArgs:       a.globArgs,
		Mapper:         a.m,
		ConvertGlobs:   true,
	}
	var current logic.Expr = nil
	var value bzl.Expr = nil
	for _, falsey := range a.featureNames {
		if current == nil {
			current = &logic.Not{&logic.Identifier{falsey}}
		} else {
			current = &logic.And{
				[]logic.Expr{
					current,
					&logic.Not{&logic.Identifier{falsey}},
				},
			}
		}
	}

	current = logic.Flatten(current)
	currentLabel := a.lb.RequestLogicalExpr(a.rel, current)

	if a.base != nil {
		value = t.Transform(a.base.Value)
	} else {
		switch a.featureProps[a.featureNames[0]].Value.Type() {
		case parser.BoolType:
			value = &bzl.LiteralExpr{Token: "False"}
		case parser.StringType:
			value = &bzl.LiteralExpr{Token: ""}
		case parser.Int64Type:
			value = &bzl.LiteralExpr{Token: "0"}
		case parser.ListType:
			value = &bzl.LiteralExpr{Token: "[]"}
		}

	}

	return &bzl.KeyValueExpr{
		Key:   &bzl.StringExpr{Value: currentLabel.String()},
		Value: value,
	}
}

func (a *Selective) createCombination(combination int) *bzl.KeyValueExpr {
	t := &bp2bzl.Transformer{
		ResolveTargets: true,
		GlobArgs:       a.globArgs,
		Mapper:         a.m,
		ConvertGlobs:   true,
	}
	var current logic.Expr = nil
	var value bzl.Expr = nil

	value = t.Transform(a.featureProps[a.featureNames[combination]].Value)

	current = &logic.Identifier{a.featureNames[combination]}
	for _, falsy := range a.featureNames[combination+1:] {
		current = &logic.And{
			[]logic.Expr{
				current,
				&logic.Not{&logic.Identifier{falsy}},
			},
		}
	}

	current = logic.Flatten(current)
	currentLabel := a.lb.RequestLogicalExpr(a.rel, current)

	return &bzl.KeyValueExpr{
		Key:   &bzl.StringExpr{Value: currentLabel.String()},
		Value: value,
	}
}

func (a *Selective) BzlExpr() bzl.Expr {

	t := &bp2bzl.Transformer{
		ResolveTargets: true,
		GlobArgs:       a.globArgs,
		Mapper:         a.m,
		ConvertGlobs:   true,
	}

	if a.base == nil && len(a.featureNames) == 0 {
		panic(fmt.Sprintf("Impossible attribute state; %#v", a))
	}

	if len(a.featureNames) == 0 && a.base != nil {
		return t.Transform(a.base.Value)
	}

	args := []*bzl.KeyValueExpr{}

	args = append(args, a.createBaseCase())
	for i := 0; i != len(a.featureNames); i++ {
		args = append(args, a.createCombination(i))
	}

	return &bzl.CallExpr{
		X:    &bzl.Ident{Name: "select"},
		List: []bzl.Expr{&bzl.DictExpr{List: args, ForceMultiLine: true}},
	}
}

func (a *Selective) Merge(other bzl.Expr) bzl.Expr      { return a.BzlExpr() }
func (a *Selective) FromName() string                   { return a.from }
func (a *Selective) ToName() string                     { return a.to }
func (a *Selective) SetValue(property *parser.Property) { a.base = property }
func (a *Selective) SetFeatureValue(name string, property *parser.Property) {
	a.featureNames = append(a.featureNames, name)
	a.featureProps[name] = property
}
func (a *Selective) SetGlobArgs(args []bzl.Expr) { a.globArgs = args }
