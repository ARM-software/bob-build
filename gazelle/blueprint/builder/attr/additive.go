// Provides support for Bob attributes which are featurable and additive in nature.
// That is, their values can be merged together (arrays)
// Given the following module:
//```
//bob_module {
//	attr: <base_value>,
//	<feature0>: {
//		attr: <feature0_value>,
//	},
//	...
//	<featureN>: {
//		attr: <featureN_value>,
//	}
//}
//```
// The expected Bazel attribute would be:
//```
//bob_module(
//	attr' = <base_value> +
//		select({
//			"//conditions:default": [],
//			"//:feature0_label": <feature0_value>,
//		})
//		+
//		...
//		+
//		select({
//			"//conditions:default": [],
//			"//:featureN_label": <featureN_value>,
//		}),
//)
//```

package attr

import (
	"github.com/ARM-software/bob-build/gazelle/blueprint/builder/bp2bzl"
	"github.com/ARM-software/bob-build/gazelle/common"
	"github.com/ARM-software/bob-build/gazelle/logic"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

type Additive struct {
	from, to string // Source and destination attribute names, allows mapping from Bob attr name to Bazel attr name
	m        *mapper.Mapper

	globArgs     []bzl.Expr
	base         *parser.Property
	featureProps map[string]*parser.Property
	featureNames []string
}

var _ Attribute = (*Additive)(nil) // impl check

func (a *Additive) BzlExpr() bzl.Expr {
	t := &bp2bzl.Transformer{
		ResolveTargets: true,
		GlobArgs:       a.globArgs,
		Mapper:         a.m,
		ConvertGlobs:   true,
	}

	ret := t.Transform(a.base.Value)

	for _, featureName := range a.featureNames {
		featureLabel := a.m.FromValue(&logic.Identifier{featureName})
		labelSelect := &bzl.CallExpr{
			X: &bzl.Ident{Name: "select"},
			List: []bzl.Expr{
				&bzl.DictExpr{
					List: []*bzl.KeyValueExpr{
						// Empty default
						{
							Key:   &bzl.StringExpr{Value: common.ConditionDefault},
							Value: &bzl.ListExpr{},
						},
						{
							Key:   &bzl.StringExpr{Value: featureLabel.String()},
							Value: t.Transform(a.featureProps[featureName].Value),
						},
					},
					ForceMultiLine: true,
				},
			},
		}

		if ret == nil {
			ret = labelSelect
		} else {
			ret = &bzl.BinaryExpr{
				X:  ret,
				Y:  labelSelect,
				Op: "+"}
		}
	}

	return ret
}

func (a *Additive) Merge(other bzl.Expr) bzl.Expr      { return a.BzlExpr() }
func (a *Additive) FromName() string                   { return a.from }
func (a *Additive) ToName() string                     { return a.to }
func (a *Additive) SetValue(property *parser.Property) { a.base = property }
func (a *Additive) SetFeatureValue(name string, property *parser.Property) {
	a.featureNames = append(a.featureNames, name)
	a.featureProps[name] = property
}
func (a *Additive) SetGlobArgs(args []bzl.Expr) { a.globArgs = args }
