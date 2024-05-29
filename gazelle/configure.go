package plugin

import (
	"os"
	"path/filepath"

	pluginConfig "github.com/ARM-software/bob-build/gazelle/config"
	"github.com/ARM-software/bob-build/gazelle/util"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	BobRootDirective = "bob_root" // Directive used to mark the root module of the Bob workspace
	ExcludeDirective = "exclude"
)

// KnownDirectives returns a list of directive keys that this Configurer can
// interpret. Gazelle prints errors for directives that are not recognized by
// any Configurer.
func (e *BobExtension) KnownDirectives() []string {
	return []string{
		BobRootDirective,
		ExcludeDirective,
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
	if _, exists := c.Exts[BobExtensionName]; !exists {
		rootCfg := pluginConfig.NewRootConfig(c.RepoRoot)
		c.Exts[BobExtensionName] = pluginConfig.ConfigMap{"": rootCfg}
		rootCfg.Blueprint.Builder.ConfigureDefault()
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
			case ExcludeDirective:
				pc.BobIgnoreDir = append(pc.BobIgnoreDir, d.Value)
			}
		}
	}

	for _, ignored := range pc.BobIgnoreDir {
		if isChild, _ := util.IsChildFilepath(ignored, rel); isChild {
			pc.IsIgnored = true
		}
	}

	// Parse ASTs
	if dir, err := os.Open(filepath.Join(c.RepoRoot, rel)); err == nil {
		if files, err := dir.ReadDir(0); err == nil {
			for _, file := range files {
				if file.Type().IsDir() {
					continue
				}

				if util.Contains(pc.Mconfig.Filenames, file.Name()) {
					pc.Files[file.Name()], err = pc.Mconfig.Parser.Parse(c.RepoRoot, rel, file.Name())
				} else if util.Contains(pc.Blueprint.Filenames, file.Name()) {
					pc.Files[file.Name()], err = pc.Blueprint.Parser.Parse(c.RepoRoot, rel, file.Name())
				}
			}
		} else {
			panic(err)
		}
	} else {
		panic(err)
	}
}
