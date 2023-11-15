// Helper module to represent the intermediate struct used to generate
// Bazel attributes from Bob AST
// Each attribute type must conform to the `Attribute` interface.
// There are currently 3 types of attributes in Bob:
// - immutable, see immutable.go
// - featurable:additive, see additive.go
// - featurable:selective,  see seleective.go
//
// This allows the user to configure the parser via the following directive syntax:
//```
//#gazelle:bob_attr <BobType> <AttrType> <From> <To>
//```
// For example:
//#gazelle:bob_attr bob_library additive srcs

package attr

import (
	"fmt"

	lb "github.com/ARM-software/bob-build/gazelle/logic/builder"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
	"github.com/google/blueprint/parser"
)

type AttrType string

const (
	AttrTypeImmutable AttrType = "immutable"
	AttrTypeAdditive           = "additive"
	AttrTypeSelective          = "selective"
	AttrTypeIgnored            = "ignore"
)

type FromType string
type ToType string

type Attribute interface {
	rule.Merger
	rule.BzlExprValue
	FromName() string // From attribute name, aka from Bob attribute name
	ToName() string   // To attribute name, aka the Bazel attribute name
	SetValue(property *parser.Property)
	SetFeatureValue(name string, property *parser.Property)
	SetGlobArgs(args []bzl.Expr)
}

// Factory function to create attribute structs
func NewAttribute(from, to string, t AttrType, m *mapper.Mapper, lb *lb.Builder) Attribute {
	if to == "" {
		to = from
	}

	switch t {
	case AttrTypeIgnored:
		return &Ignored{from: from, to: to}
	case AttrTypeImmutable:
		return &Immutable{
			from: from,
			to:   to}
	case AttrTypeAdditive:
		return &Additive{from: from, to: to, m: m, featureProps: map[string]*parser.Property{}}
	case AttrTypeSelective:
		return &Selective{from: from, to: to, m: m, featureProps: map[string]*parser.Property{}, lb: lb}
	default:
		panic(fmt.Sprintf("Unsupported attribute type requested: '%s' from:'%s' to: '%s'", t, from, to))
	}
}
