// The logic package takes labels to configurations creates/returns rules to achieve the desired expression.
// For example `//:config` points to a Mconfig configuration.
// Using logic we can express the following:
// - And(labels...)
// - Or(labels...)
// - Not(label) / False(label)
// - True(label)
//
// In Bazel most configurations are done via select, complex expressions can be done by combining multiple `selects.config_setting_group`.
// The challenge is to emit the interim rules and produce a required rule/label given an expression. For example given configA and configB,
// we can convert And(True(configA), False(configB)) will produce:
//```
// config_setting(
//     name = "_configA_true",
//     values = {":configA": true},
// )
// config_setting(
//     name = "_configB_false",
//     values = {":configB": false},
// )
// selects.config_setting_group(
//     name = "_configA_true_and_configB_false",
//     match_all = [":_configA_true", ":_configB_false"],
// )
//```

package logic

import (
	"fmt"
	"strings"
)

type Type uint

const (
	NotType Type = iota + 1
	TrueType
	FalseType
	AndType
	OrType
	IdentifierType //
)

type Expr interface {
	Type() Type
	String() string
	// Equal(Expr) bool
}

// An identifier can hold abstract value of any Type.
type Identifier struct {
	Value interface{}
}

func (e *Identifier) Type() Type     { return IdentifierType }
func (e *Identifier) String() string { return fmt.Sprintf("[%v]", e.Value) }

type Not struct {
	Value Expr
}

func (e *Not) Type() Type     { return NotType }
func (e *Not) String() string { return fmt.Sprintf("!%s", e.Value.String()) }

type And struct {
	Values []Expr
}

func (e *And) Type() Type { return AndType }
func (e *And) String() string {
	vs := make([]string, len(e.Values))
	for i, value := range e.Values {
		vs[i] = value.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(vs, "*"))
}

type Or struct {
	Values []Expr
}

func (e *Or) String() string {
	vs := make([]string, len(e.Values))
	for i, value := range e.Values {
		vs[i] = value.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(vs, "+"))
}

func (e *Or) Type() Type { return OrType }

func NewIdentifier(value interface{}) Expr {
	return &Identifier{Value: value}
}

func NewExpr(t Type, args ...Expr) Expr {
	switch t {

	case NotType:
		return &Not{Value: args[0]}
	case AndType:
		return &And{Values: args}
	case OrType:
		return &Or{Values: args}

	case IdentifierType:
		return args[0]
	}
	return nil
}
