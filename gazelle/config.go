package plugin

import (
	"log"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KnownDirectives returns a list of directive keys that this Configurer can
// interpret. Gazelle prints errors for directives that are not recoginized by
// any Configurer.
func (e *BobExtension) KnownDirectives() []string {
	log.Printf("KnownDirectives() - NOT IMPLEMENTED\n")
	return nil
}

// Configure modifies the configuration using directives and other information
// extracted from a build file. Configure is called in each directory.
//
// c is the configuration for the current directory. It starts out as a copy
// of the configuration for the parent directory.
//
// rel is the slash-separated relative path from the repository root to
// the current directory. It is "" for the root directory itself.
//
// f is the build file for the current directory or nil if there is no
// existing build file.
func (e *BobExtension) Configure(c *config.Config, rel string, f *rule.File) {
	log.Printf("Configure() - NOT IMPLEMENTED\n")
}
