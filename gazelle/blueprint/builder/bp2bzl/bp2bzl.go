package bp2bzl

import (
	"fmt"
	"strings"

	"github.com/ARM-software/bob-build/gazelle/mapper"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

type Transformer struct {
	ResolveTargets bool
	ConvertGlobs   bool
	GlobArgs       []bzl.Expr
	Mapper         *mapper.Mapper
}

func (t *Transformer) isGlobValue(expr parser.Expression) bool {
	switch e := expr.(type) {
	case *parser.String:
		return strings.Contains(e.Value, "*")
	default:
		return false
	}
}

func (t *Transformer) Transform(expr parser.Expression) bzl.Expr {
	switch e := expr.(type) {
	case *parser.Int64:
		return &bzl.LiteralExpr{Token: fmt.Sprintf("%d", e.Value)}
	case *parser.Bool:
		token := "False"
		if e.Value {
			token = "True"
		}
		return &bzl.LiteralExpr{Token: token}

	case *parser.String:
		if t.ResolveTargets {
			label := t.Mapper.FromValue(e.Value)
			if label != nil {
				return &bzl.StringExpr{Value: label.String()}
			}
		}
		return &bzl.StringExpr{Value: e.Value}

	case *parser.List:
		var list []bzl.Expr
		var globs []string

		for _, elem := range e.Values {
			if t.ConvertGlobs && t.isGlobValue(elem) {
				globs = append(globs, elem.(*parser.String).Value)
			} else {

				list = append(list, t.Transform(elem))
			}
		}

		var current bzl.Expr = nil

		for _, glob := range globs {
			globArgs := []bzl.Expr{
				rule.ExprFromValue([]string{glob}),
			}

			globArgs = append(globArgs, t.GlobArgs...)

			globExpr := &bzl.CallExpr{
				X:    &bzl.LiteralExpr{Token: "glob"},
				List: globArgs,
			}
			if current == nil {
				current = globExpr
			} else {
				current = &bzl.BinaryExpr{
					X:  current,
					Y:  globExpr,
					Op: "+",
				}
			}
		}

		if current == nil {
			current = &bzl.ListExpr{List: list}
		} else if len(list) > 0 {
			current = &bzl.BinaryExpr{
				X:  &bzl.ListExpr{List: list},
				Y:  current,
				Op: "+"}
		}

		return current

	case *parser.Operator:
		return &bzl.BinaryExpr{
			X:  t.Transform(e.Args[0]),
			Y:  t.Transform(e.Args[1]),
			Op: fmt.Sprintf("%s", e.Operator),
		}
	default:
		fmt.Printf("Unhandled type %#v\n", expr)
	}
	return nil
}
