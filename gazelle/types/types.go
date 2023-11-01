package types

import (
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
