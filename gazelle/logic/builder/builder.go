package builder

import (
	"github.com/ARM-software/bob-build/gazelle/logic"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	"github.com/ARM-software/bob-build/gazelle/util"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
	bzl "github.com/bazelbuild/buildtools/build"
)

type LogicalRule interface {
	Generate() language.GenerateResult
}

type Match struct {
	matchType string
	expr      logic.Expr
	labels    []*label.Label
}

func (m *Match) Generate() (result language.GenerateResult) {
	r := rule.NewRule("selects.config_setting_group", m.expr.String())
	args := []bzl.Expr{}

	for _, l := range m.labels {
		args = append(args, rule.ExprFromValue(l.String()))
	}

	r.SetAttr(m.matchType, args)

	result.Gen = append(result.Gen, r)
	result.Imports = append(result.Imports, "")
	return
}

type Builder struct {
	m        *mapper.Mapper
	requests map[string][]LogicalRule
}

func New(m *mapper.Mapper) *Builder {
	return &Builder{
		m:        m,
		requests: map[string][]LogicalRule{},
	}
}

func (b *Builder) Build(args language.GenerateArgs) (result language.GenerateResult) {

	for _, r := range b.requests[args.Rel] {
		result = util.MergeResults(result, r.Generate())
	}

	return
}

func (b *Builder) RequestLogicalExpr(rel string, expr logic.Expr) *label.Label {

	l := b.m.FromValue(expr)

	if l != nil {
		return l
	}

	l = mapper.MakeLabel(expr.String(), rel)
	b.m.Map(l, expr.String())

	switch expr := expr.(type) {
	case *logic.And:
		matchLabels := []*label.Label{}
		for _, v := range expr.Values {
			c := b.RequestLogicalExpr(rel, v)
			if c != nil {
				matchLabels = append(matchLabels, c)
			}
		}
		b.requests[rel] = append(b.requests[rel], &Match{"match_all", expr, matchLabels})
	case *logic.Or:
		matchLabels := []*label.Label{}
		for _, v := range expr.Values {
			c := b.RequestLogicalExpr(rel, v)
			if c != nil {
				matchLabels = append(matchLabels, c)
			}
		}
		b.requests[rel] = append(b.requests[rel], &Match{"match_any", expr, matchLabels})
	}

	return l
}
