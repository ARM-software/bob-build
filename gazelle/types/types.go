package types

import (
	"sort"
	"strings"

	"github.com/ARM-software/bob-build/gazelle/common"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

// SrcsAttribute represents `srcs` attribute which
// can contain complex expression with `glob` & `select`.
//
// It implements `rule.Merger` interface thus can be used
// with custom merge algorithm.
//
// Main purpose is to avoid Gazelle drawback where merging
// `select` is sticked with defined platforms only:
// https://github.com/bazelbuild/bazel-gazelle/issues/1051
type SrcsAttribute struct {
	Expr bzl.Expr
}

var _ rule.BzlExprValue = (*SrcsAttribute)(nil)
var _ rule.Merger = (*SrcsAttribute)(nil)

func (m *SrcsAttribute) Merge(other bzl.Expr) bzl.Expr {
	// TODO: implement custom merging
	// For now return newly generated expression.
	// It means it will replace existing expression.
	return m.Expr
}

func (m SrcsAttribute) BzlExpr() bzl.Expr {
	return m.Expr
}

type GlobValue struct {
	Patterns           []string
	Excludes           []string
	ExcludeDirectories *bool
	AllowEmpty         *bool
}

type StringListWithGlob struct {
	list    []string
	glob    GlobValue
	hasGlob bool
}

func MakeStringListWithGlob(list []string) *StringListWithGlob {

	s := &StringListWithGlob{}

	if list == nil || len(list) == 0 {
		return nil
	}

	for _, l := range list {
		if strings.Contains(l, "*") {
			s.glob.Patterns = append(s.glob.Patterns, l)
			s.hasGlob = true
		} else {
			s.list = append(s.list, l)
		}
	}

	return s
}

func (s *StringListWithGlob) BzlExpr() bzl.Expr {

	list := make([]bzl.Expr, 0, 2)

	if s.list != nil && len(s.list) > 0 {
		val := rule.ExprFromValue(s.list)
		if v, ok := val.(*bzl.ListExpr); ok {
			v.ForceMultiLine = true
		}
		list = append(list, val)
	}

	if s.hasGlob {
		list = append(list, s.glob.BzlExpr())
	}

	if len(list) == 2 {
		return &bzl.BinaryExpr{X: list[0], Y: list[1], Op: "+"}
	} else if len(list) == 1 {
		return list[0]
	}

	return nil
}

func (s *GlobValue) BzlExpr() bzl.Expr {

	patternsValue := rule.ExprFromValue(s.Patterns)
	globArgs := []bzl.Expr{patternsValue}

	if s.AllowEmpty != nil {
		allowEmptyValue := rule.ExprFromValue(*s.AllowEmpty)
		globArgs = append(globArgs, &bzl.AssignExpr{
			LHS: &bzl.LiteralExpr{Token: "allow_empty"},
			Op:  "=",
			RHS: allowEmptyValue,
		})
	}

	if len(s.Excludes) > 0 {
		excludesValue := rule.ExprFromValue(s.Excludes)
		globArgs = append(globArgs, &bzl.AssignExpr{
			LHS: &bzl.LiteralExpr{Token: "exclude"},
			Op:  "=",
			RHS: excludesValue,
		})
	}

	if s.ExcludeDirectories != nil {
		excludeDirValue := &bzl.LiteralExpr{Token: "1"}
		if !(*s.ExcludeDirectories) {
			excludeDirValue = &bzl.LiteralExpr{Token: "0"}
		}

		globArgs = append(globArgs, &bzl.AssignExpr{
			LHS: &bzl.LiteralExpr{Token: "exclude_directories"},
			Op:  "=",
			RHS: excludeDirValue,
		})
	}

	return &bzl.CallExpr{
		X:    &bzl.LiteralExpr{Token: "glob"},
		List: globArgs,
	}
}

type SelectStringListWithGlob map[string]*StringListWithGlob

func (s SelectStringListWithGlob) BzlExpr() bzl.Expr {
	keys := make([]string, 0, len(s))
	hasDefaultKey := false

	for key := range s {
		if key == common.ConditionDefault {
			hasDefaultKey = true
		} else {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	args := make([]*bzl.KeyValueExpr, 0, len(s))

	for _, key := range keys {
		value := s[key].BzlExpr()

		args = append(args, &bzl.KeyValueExpr{
			Key:   &bzl.StringExpr{Value: key},
			Value: value,
		})
	}

	var v bzl.Expr

	// ConditionDefault in the end of the `select`
	if hasDefaultKey {
		v = s[common.ConditionDefault].BzlExpr()
	} else {
		// empty '//conditions:default'
		v = rule.ExprFromValue([]string{})
	}

	args = append(args, &bzl.KeyValueExpr{
		Key:   &bzl.StringExpr{Value: common.ConditionDefault},
		Value: v,
	})

	sel := &bzl.CallExpr{
		X:    &bzl.Ident{Name: "select"},
		List: []bzl.Expr{&bzl.DictExpr{List: args, ForceMultiLine: true}},
	}

	return sel
}

type Generator interface {
	GenerateRule() (*rule.Rule, error)
}
