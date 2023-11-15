package attr

import (
	"github.com/ARM-software/bob-build/gazelle/blueprint/builder/bp2bzl"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

// Immutable attributes ignore features completely.
// These are essentially plain value types.
type Immutable struct {
	from, to string // Source and destination attribute names, allows mapping from Bob attr name to Bazel attr name
	value    *parser.Property
}

var _ Attribute = (*Immutable)(nil) // impl check

func (a *Immutable) BzlExpr() bzl.Expr {
	t := &bp2bzl.Transformer{ResolveTargets: false}
	return t.Transform(a.value.Value)
}

func (a *Immutable) Merge(other bzl.Expr) bzl.Expr      { return other }
func (a *Immutable) FromName() string                   { return a.from }
func (a *Immutable) ToName() string                     { return a.to }
func (a *Immutable) SetValue(property *parser.Property) { a.value = property }

// Not implemented, features are not supported
func (a *Immutable) SetFeatureValue(string, *parser.Property) {}

func (a *Immutable) SetGlobArgs(args []bzl.Expr) {}
