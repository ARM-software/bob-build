package config

import (
	"path/filepath"
)

// Config stores the plugin configuration, it is created for each directory in which the plugin runs.
// The structure is designed to inherit parent configuration to propagate settings on a tree level.
type Config struct {
	parent                  *Config
	RepositoryRootPath      string   // Absolute path to the root of the workspace.
	BobWorkspaceRootRelPath string   // Relative module path to Bob workspace root.
	BobIgnoreDir            []string // Relative path to ignore list from workspace root
}

func (c *Config) NewChild() *Config {
	return &Config{
		parent:                  c,
		RepositoryRootPath:      c.RepositoryRootPath,
		BobWorkspaceRootRelPath: c.BobWorkspaceRootRelPath,
		BobIgnoreDir:            c.BobIgnoreDir,
	}
}

func NewRootConfig(repositoryRootPath string) *Config {
	return &Config{
		parent:                  nil,
		RepositoryRootPath:      repositoryRootPath,
		BobWorkspaceRootRelPath: "",
		BobIgnoreDir:            nil,
	}
}

// Map of configs keyed on relative module paths.
type ConfigMap map[string]*Config

func (c *ConfigMap) ParentForModulePath(relativeModulePath string) *Config {
	dir := filepath.Dir(relativeModulePath)
	if dir == "." {
		dir = ""
	}
	parent := (map[string]*Config)(*c)[dir]
	return parent
}
