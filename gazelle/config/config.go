package config

import (
	"path/filepath"

	bb "github.com/ARM-software/bob-build/gazelle/blueprint/builder"
	bp "github.com/ARM-software/bob-build/gazelle/blueprint/parser"
	lb "github.com/ARM-software/bob-build/gazelle/logic/builder"
	"github.com/ARM-software/bob-build/gazelle/mapper"
	mb "github.com/ARM-software/bob-build/gazelle/mconfig/builder"
	mp "github.com/ARM-software/bob-build/gazelle/mconfig/parser"
)

type MconfigConfig struct {
	Parser    *mp.Parser
	Builder   *mb.Builder
	Filenames []string // Filenames to parse within a directory
}
type BlueprintConfig struct {
	Parser    *bp.Parser
	Builder   *bb.Builder
	Filenames []string // Filenames to parse within a directory
}
type LogicConfig struct {
	Builder *lb.Builder
}

// Config stores the plugin configuration, it is created for each directory in which the plugin runs.
// The structure is designed to inherit parent configuration to propagate settings on a tree level.
type Config struct {
	parent                  *Config
	RepositoryRootPath      string   // Absolute path to the root of the workspace.
	BobWorkspaceRootRelPath string   // Relative module path to Bob workspace root.
	BobIgnoreDir            []string // Relative path to ignore list from workspace root

	IsIgnored bool
	Mapper    *mapper.Mapper
	Blueprint BlueprintConfig
	Mconfig   MconfigConfig
	Logic     LogicConfig            // Configuration for generating logical expressions.
	Files     map[string]interface{} // Parsed file AST

}

func (c *Config) NewChild() *Config {
	return &Config{
		parent:                  c,
		RepositoryRootPath:      c.RepositoryRootPath,
		BobWorkspaceRootRelPath: c.BobWorkspaceRootRelPath,
		BobIgnoreDir:            c.BobIgnoreDir,
		IsIgnored:               c.IsIgnored,

		Blueprint: BlueprintConfig{
			// Pass parent scope for blueprint parsing
			Parser: bp.New(c.Mapper, c.Blueprint.Parser.GetScope()),

			// Inherit other settings
			Filenames: c.Blueprint.Filenames,
			Builder:   c.Blueprint.Builder,
		},
		Mconfig: c.Mconfig,
		Logic:   c.Logic,
		Mapper:  c.Mapper,
		Files:   map[string]interface{}{},
	}
}

func NewRootConfig(repositoryRootPath string) *Config {
	m := mapper.NewMapper()
	lb := lb.New(m)
	return &Config{
		RepositoryRootPath: repositoryRootPath,
		Mconfig: MconfigConfig{
			Filenames: []string{"Mconfig"}, //Default filenames to parse
			Builder:   mb.NewBuilder(m, lb),
			Parser:    mp.New(m),
		},
		Blueprint: BlueprintConfig{
			Filenames: []string{"build.bp"}, //Default filenames to parse
			Builder:   bb.NewBuilder(m, lb),
			Parser:    bp.New(m, nil),
		},
		Logic:     LogicConfig{lb},
		Files:     map[string]interface{}{},
		Mapper:    m,
		IsIgnored: false,
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
