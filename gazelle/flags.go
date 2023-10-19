package plugin

import (
	"flag"

	"github.com/bazelbuild/bazel-gazelle/config"
)

// RegisterFlags registers command-line flags used by the Bobextension. This
// method is called once with the root configuration when Gazelle
// starts. RegisterFlags may set an initial values in Config.Exts. When flags
// are set, they should modify these values.
func (e *BobExtension) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
}

// CheckFlags validates the configuration after command line flags are parsed.
// This is called once with the root configuration when Gazelle starts.
// CheckFlags may set default values in flags or make implied changes.
func (e *BobExtension) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}
