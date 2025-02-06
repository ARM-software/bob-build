package backend

import (
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

type AndroidNinjaPlatform struct {
	toolchains toolchain.ToolchainSet
	env        *config.EnvironmentVariables
}

// BinaryOutputDir implements Platform.
func (*AndroidNinjaPlatform) BinaryOutputDir(tgt toolchain.TgtType) string {
	panic("unimplemented")
}

// BobScriptsDir implements Platform.
func (*AndroidNinjaPlatform) BobScriptsDir() string {
	panic("unimplemented")
}

// BuildDir implements Platform.
func (*AndroidNinjaPlatform) BuildDir() string {
	panic("unimplemented")
}

// EscapeFlag implements Platform.
func (*AndroidNinjaPlatform) EscapeFlag(string) string {
	panic("unimplemented")
}

// GetToolchain implements Platform.
func (*AndroidNinjaPlatform) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	panic("unimplemented")
}

// Init implements Platform.
func (*AndroidNinjaPlatform) Init(*config.Properties) {
	panic("unimplemented")
}

// KernelModOutputDir implements Platform.
func (*AndroidNinjaPlatform) KernelModOutputDir() string {
	panic("unimplemented")
}

// SharedLibsDir implements Platform.
func (*AndroidNinjaPlatform) SharedLibsDir(tgt toolchain.TgtType) string {
	panic("unimplemented")
}

// SourceDir implements Platform.
func (*AndroidNinjaPlatform) SourceDir() string {
	panic("unimplemented")
}

// SourceOutputDir implements Platform.
func (*AndroidNinjaPlatform) SourceOutputDir(blueprint.Module) string {
	panic("unimplemented")
}

// StaticLibOutputDir implements Platform.
func (*AndroidNinjaPlatform) StaticLibOutputDir(tgt toolchain.TgtType) string {
	panic("unimplemented")
}

var _ Platform = (*AndroidNinjaPlatform)(nil)
