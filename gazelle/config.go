package plugin

import (
	"log"
	"os"
	"path/filepath"

	pluginConfig "github.com/ARM-software/bob-build/gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	BobRootDirective = "bob_root" // Directive used to mark the root module of the Bob workspace
)

// KnownDirectives returns a list of directive keys that this Configurer can
// interpret. Gazelle prints errors for directives that are not recognized by
// any Configurer.
func (e *BobExtension) KnownDirectives() []string {
	return []string{
		BobRootDirective,
	}
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

	isBobRoot := false

	if _, exists := c.Exts[BobExtensionName]; !exists {
		rootCfg := pluginConfig.NewRootConfig(c.RepoRoot)
		c.Exts[BobExtensionName] = pluginConfig.ConfigMap{"": rootCfg}
	}

	configs := c.Exts[BobExtensionName].(pluginConfig.ConfigMap)

	// Get plugin configuration for this path, if none exists, create it.
	pc, exists := configs[rel]
	if !exists {
		parent := configs.ParentForModulePath(rel)
		pc = parent.NewChild()
		configs[rel] = pc
	}

	// Handle directives
	if f != nil {
		for _, d := range f.Directives {
			switch d.Key {
			case BobRootDirective:
				pc.BobWorkspaceRootRelPath = rel
				isBobRoot = true
			}
		}
	}

	if isBobRoot {
		if _, err := os.Stat(filepath.Join(c.RepoRoot, rel, "Mconfig")); err != nil {
			log.Fatalf("No root Mconfig file: %v\n", err)
		}

		fileNames := []string{"Mconfig"}

		parser := newMconfigParser(c.RepoRoot, rel)

		// TODO use returned configs from `mconfigParser.parse()`
		_, err := parser.parse(&fileNames)
		if err != nil {
			log.Fatalf("Parse failed: %v\n", err)
		}
	}
}
