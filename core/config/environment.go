package config

import (
	"os"
	"sync"
)

// Stores all environment variables passed to Bob at runtime as a singleton.
type EnvironmentVariables struct {
	BobDir          string
	SrcDir          string
	ConfigOpts      string
	ConfigFile      string
	ConfigJSON      string
	LogWarningsFile string
	LogWarnings     string
	BuildMetaFile   string
}

var env *EnvironmentVariables
var lock = &sync.Mutex{}

// Return the environment variables.
// If called for the first time, read the environment once and store it for future use.
// These are not initialized on module entry to allow gendiffer to override these for testing.
func GetEnvironmentVariables() *EnvironmentVariables {
	if env == nil {
		lock.Lock()
		defer lock.Unlock()
		if env == nil {
			env = &EnvironmentVariables{
				BobDir:          os.Getenv("BOB_DIR"),
				SrcDir:          os.Getenv("SRCDIR"),
				ConfigOpts:      os.Getenv("BOB_CONFIG_OPTS"),
				ConfigFile:      os.Getenv("CONFIG_FILE"),
				ConfigJSON:      os.Getenv("CONFIG_JSON"),
				LogWarningsFile: os.Getenv("BOB_LOG_WARNINGS_FILE"),
				LogWarnings:     os.Getenv("BOB_LOG_WARNINGS"),
				BuildMetaFile:   os.Getenv("BOB_META_FILE"),
			}
		}
	}
	return env
}
