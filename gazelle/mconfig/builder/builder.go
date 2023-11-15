package builder

import (
	"fmt"
	"log"
	"reflect"
	"sort"

	"github.com/ARM-software/bob-build/gazelle/common"
	"github.com/ARM-software/bob-build/gazelle/kinds"
	"github.com/ARM-software/bob-build/gazelle/logic"
	lb "github.com/ARM-software/bob-build/gazelle/logic/builder"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	mparser "github.com/ARM-software/bob-build/gazelle/mconfig/parser"
	"github.com/ARM-software/bob-build/gazelle/util"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

type Builder struct {
	m  *mapper.Mapper
	lb *lb.Builder
}

func NewBuilder(m *mapper.Mapper, lb *lb.Builder) *Builder {
	return &Builder{m, lb}
}

func ParseLogic(m *mapper.Mapper, expr interface{}) logic.Expr {
	rv := reflect.ValueOf(expr)
	switch rv.Kind() {
	case reflect.String:
		return logic.NewIdentifier(expr.(string))

	case reflect.Slice, reflect.Array:
		types := map[string]logic.Type{
			"or":         logic.OrType,
			"and":        logic.AndType,
			"not":        logic.NotType,
			"identifier": logic.IdentifierType,
		}

		var args []logic.Expr
		for i := 1; i < rv.Len(); i++ {
			args = append(args, ParseLogic(m, rv.Index(i).Interface()))
		}

		if t, ok := types[rv.Index(0).Interface().(string)]; ok {
			return logic.NewExpr(t, args...)
		}
	}

	return nil

}

func LiteralExpressionToBzl(expression []interface{}) bzl.Expr {
	if len(expression) != 2 {
		panic(fmt.Sprintf("Cannot convert expression '%v' to value literal", expression))
	}

	switch expression[0].(string) {
	case "boolean":
		yesno := expression[1].(bool)
		return rule.ExprFromValue(yesno)
	case "string":
		return rule.ExprFromValue(expression[1].(string))
	case "number":
		return &bzl.LiteralExpr{Token: fmt.Sprintf("%d", int(expression[1].(float64)))}
	default:
		panic(fmt.Sprintf("Cannot convert expression '%v' to value literal, unknown type", expression))
	}
}

type FlagDefaultValue struct {
	m                 *mapper.Mapper
	conditionalValues map[*label.Label]bzl.Expr
	conditionLabels   []*label.Label // TODO: switch this to key:value pairs in a list to preserve gen ordering
	defaultValue      bzl.Expr
}

func (b *Builder) NewFlagDefaultValue(rel string, c *mparser.ConfigData) *FlagDefaultValue {
	var conditionals map[*label.Label]bzl.Expr = nil
	var defaultValue bzl.Expr = nil
	conditionLabels := []*label.Label{}

	if len(c.ConditionalDefaults) > 0 {
		conditionals = map[*label.Label]bzl.Expr{}

		for _, conditional := range c.ConditionalDefaults {
			t := logic.Flatten(ParseLogic(b.m, conditional.Condition))
			l := b.lb.RequestLogicalExpr(rel, t)
			conditionLabels = append(conditionLabels, l)
			conditionals[l] = LiteralExpressionToBzl(conditional.Expression)
		}
	}

	if c.Default != nil {
		defaultValue = LiteralExpressionToBzl(c.Default)
	} else {
		switch c.Datatype {
		case "int":
			defaultValue = &bzl.LiteralExpr{Token: "0"}
		case "bool":
			defaultValue = rule.ExprFromValue(false)
		case "string":
			defaultValue = rule.ExprFromValue("")
		default:
			panic(fmt.Sprintf("Cannot handle configuration datatype %v", c.Datatype))
		}
	}

	return &FlagDefaultValue{
		m:                 b.m,
		conditionalValues: conditionals,
		conditionLabels:   conditionLabels,
		defaultValue:      defaultValue,
	}
}

func (a *FlagDefaultValue) BzlExpr() bzl.Expr {
	if a.conditionalValues == nil || len(a.conditionLabels) == 0 {
		return a.defaultValue
	}

	args := make([]*bzl.KeyValueExpr, 0, len(a.conditionalValues)+1)

	for _, label := range a.conditionLabels {
		args = append(args, &bzl.KeyValueExpr{
			Key:   rule.ExprFromValue(label.String()),
			Value: a.conditionalValues[label],
		})
	}

	args = append(args, &bzl.KeyValueExpr{
		Key:   rule.ExprFromValue(common.ConditionDefault),
		Value: a.defaultValue,
	})

	return &bzl.CallExpr{
		X:    &bzl.Ident{Name: "select"},
		List: []bzl.Expr{&bzl.DictExpr{List: args, ForceMultiLine: true}},
	}
}

func (a *FlagDefaultValue) Merge(other bzl.Expr) bzl.Expr { return a.BzlExpr() }

func getFlagLabel(m *mapper.Mapper, expr logic.Expr) *label.Label {
	switch expr := expr.(type) {
	case *logic.Not:
		return getFlagLabel(m, expr.Value)
	case *logic.Identifier:
		return m.FromValue(expr.Value)
	}

	panic(fmt.Sprintf("Unknown expression type %v!", expr))
}

// For example:
// config NEW_HW
// 	bool "New hardware platform"
// 	depends on FEATURE_B
// 	default n
// In this case would expect the following:
// ```
// bool_flag(
// 	name = "NEW_HW",
// 	build_setting_default = False,
// )
//
// config_setting( // When the flag value is false we ignore any other conditions
// 	name = "![NEW_HW]",
// 	flag_values = {"//:NEW_HW": false},
// )
//
// config_setting(
// 	name = "__[NEW_HW_FLAG]", // interim config setting which does not account for `depends``
// 	flag_values = {"//:NEW_HW": true},
// )

// selects.config_setting_group(
// 	name = "([NEW_HW]*[FEATURE_B])",
// 	match_all = [
// 	    "//internal:__[NEW_HW_FLAG]",
// 	    "//internal:[FEATURE_B]",
// 	],
// )

// alias(
// 	name = "[NEW_HW]",
// 	actual = ":([NEW_HW]*[FEATURE_B])",
// )

// ```

func (b *Builder) generateBoolFlagSettings(args language.GenerateArgs, v *mparser.ConfigData) (result language.GenerateResult) {

	depends := logic.Flatten(ParseLogic(b.m, v.Depends))
	flagLabel := b.m.FromValue(v.Name)

	// Falsey rule, this is the same for both.
	falsyExpr := &logic.Not{&logic.Identifier{v.Name}}
	falsy := rule.NewRule("config_setting", falsyExpr.String())
	falsy.SetAttr("flag_values",
		&bzl.DictExpr{
			List: []*bzl.KeyValueExpr{
				{
					Key:   &bzl.StringExpr{Value: flagLabel.String()},
					Value: &bzl.LiteralExpr{Token: "False"},
				},
			},
		},
	)

	// Truthy rule
	truthyExpr := &logic.Identifier{fmt.Sprintf("__%s", v.Name)}
	truthy := rule.NewRule("config_setting", truthyExpr.String())
	truthy.SetAttr("flag_values",
		&bzl.DictExpr{
			List: []*bzl.KeyValueExpr{
				{
					Key:   &bzl.StringExpr{Value: flagLabel.String()},
					Value: &bzl.LiteralExpr{Token: "True"},
				},
			},
		},
	)

	withDepends := rule.NewRule("alias", (&logic.Identifier{v.Name}).String())
	if depends != nil {
		// TODO: request a config setting and remap truthy label to &logic.And{&logic.Identifier{v.Name}, depends}
		// TODO: This needs to happen in Mconfig parse such that it is done before any blueprint generation
		depends = logic.Flatten(&logic.And{[]logic.Expr{truthyExpr, depends}})
		dependsLabel := b.lb.RequestLogicalExpr(args.Rel, depends)
		withDepends.SetAttr("actual", dependsLabel.String())
		// b.m.Map(l, depends) // remap the True expression to use the depends logic
		// fmt.Printf("depends: %v\n", depends)
	} else {
		withDepends.SetAttr("actual", fmt.Sprintf(":%s", truthyExpr))
	}

	result.Imports = append(result.Imports, "", "", "")
	result.Gen = append(result.Gen, truthy, falsy, withDepends)
	return
}

func (b *Builder) Build(args language.GenerateArgs, file interface{}) (result language.GenerateResult) {
	// The builder only generates the flag targets for each Mconfig.
	// The actual configuration settings are generated by the logic module.

	// The current implementation takes a module registry pointer. In the future we may wish to change this to only
	// take the current file AST `file` and a mapper module for resolving/registering targets called `mapper`
	configs := file.(*map[string]*mparser.ConfigData)

	rules := make([]*rule.Rule, 0)

	// To properly test generation of multiple configs
	// at once the order needs to be preserved
	// Sort all configs by its Position
	configsToGen := make([]*mparser.ConfigData, 0)

	for _, cfg := range *configs {
		if cfg.Ignore != "y" {
			configsToGen = append(configsToGen, cfg)
		}
	}

	sort.Slice(configsToGen, func(i, j int) bool {
		return configsToGen[i].Position < configsToGen[j].Position
	})

	for _, v := range configsToGen {

		// v.Datatype should be one of ["bool", "string", "int"]
		if !utils.Contains([]string{"bool", "string", "int"}, v.Datatype) {
			log.Printf("Unsupported config of type '%s'", v.Datatype)
			break
		}

		ruleName := fmt.Sprintf("%s_flag", v.Datatype)
		flag := rule.NewRule(ruleName, v.Name)

		flag.SetAttr("build_setting_default", b.NewFlagDefaultValue(args.Rel, v))

		// Only bool is supported currently for flag values.
		if v.Datatype == "bool" {
			result = util.MergeResults(result, b.generateBoolFlagSettings(args, v))
		}
		rules = append(rules, flag)
	}

	for _, r := range rules {
		if r.IsEmpty(kinds.Kinds[r.Kind()]) {
			result.Empty = append(result.Empty, r)
		} else {
			result.Gen = append(result.Gen, r)
			result.Imports = append(result.Imports, r.PrivateAttr(""))
		}
	}

	return
}
