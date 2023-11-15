package attr

import (
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

type Ignored struct {
	from, to string // Source and destination attribute names, allows mapping from Bob attr name to Bazel attr name
}

var _ Attribute = (*Ignored)(nil) // impl check

func (a *Ignored) BzlExpr() bzl.Expr                                      { return nil }
func (a *Ignored) Merge(other bzl.Expr) bzl.Expr                          { return nil }
func (a *Ignored) FromName() string                                       { return a.from }
func (a *Ignored) ToName() string                                         { return a.to }
func (a *Ignored) SetValue(property *parser.Property)                     {}
func (a *Ignored) SetFeatureValue(name string, property *parser.Property) {}
func (a *Ignored) SetGlobArgs(args []bzl.Expr)                            {}
