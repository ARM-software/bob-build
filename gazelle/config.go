package plugin

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	bob "github.com/ARM-software/bob-build/core"

	pluginConfig "github.com/ARM-software/bob-build/gazelle/config"
	"github.com/ARM-software/bob-build/gazelle/util"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const (
	BobRootDirective      = "bob_root" // Directive used to mark the root module of the Bob workspace
	BobIgnoreDirDirective = "bob_ignore"
)

// KnownDirectives returns a list of directive keys that this Configurer can
// interpret. Gazelle prints errors for directives that are not recognized by
// any Configurer.
func (e *BobExtension) KnownDirectives() []string {
	return []string{
		BobRootDirective,
		BobIgnoreDirDirective,
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
			case BobIgnoreDirDirective:
				pc.BobIgnoreDir = append(pc.BobIgnoreDir, d.Value)
			}
		}
	}

	for _, ignored := range pc.BobIgnoreDir {
		if isChild, _ := util.IsChildFilepath(ignored, rel); isChild {
			isBobRoot = false // This path is in the ignored list
		}
	}

	if isBobRoot {

		if _, err := os.Stat(filepath.Join(c.RepoRoot, rel, "Mconfig")); err != nil {
			log.Fatalf("No root Mconfig file: %v\n", err)
		}
		fileNames := []string{"Mconfig"}

		mconfigParser := newMconfigParser(c.RepoRoot, rel)
		configs, err := mconfigParser.parse(&fileNames)
		if err != nil {
			log.Fatalf("Mconfig parse failed: %v\n", err)
		}

		// Register all `configData`s
		for _, c := range *configs {
			e.registry.Register(c)
		}

		bobConfig := createBobConfigSpoof(configs)
		bobParser := newBobParser(c.RepoRoot, rel, pc.BobIgnoreDir, bobConfig)

		modules := bobParser.parse()

		// Register all `Module`s
		for _, m := range modules {
			e.registry.Register(m)
			m.SetRegistry(e.registry)
		}
	}
}

func createBobConfigSpoof(c *map[string]*configData) *bob.BobConfig {

	config := &bob.BobConfig{}

	// prepare feature list
	config.Properties.FeatureList = make([]string, 0)
	config.Properties.Features = make(map[string]bool)
	config.Properties.Properties = make(map[string]interface{})

	config.Properties.Properties["osx"] = bool(false) // shared lib factory requires this.

	for k, v := range *c {
		if v.Ignore != "y" {
			config.Properties.FeatureList = append(config.Properties.FeatureList, strings.ToLower(k))
			// To be safe set everything to false by default.
			config.Properties.Features[k] = false
			config.Properties.Properties[k] = v
		}
	}

	return config
}
