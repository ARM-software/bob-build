package plugin

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
)

type generator interface {
	generateRule() (*rule.Rule, error)
}
