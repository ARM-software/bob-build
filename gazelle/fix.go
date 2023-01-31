package plugin

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix repairs deprecated usage of language-specific rules in f. This is
// called before the file is indexed. Unless c.ShouldFix is true, fixes
// that delete or rename rules should not be performed.
func (e *BobExtension) Fix(c *config.Config, f *rule.File) {
	log.Printf("Fix() - NOT IMPLEMENTED\n")
}
