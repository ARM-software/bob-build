package backend

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

type AndroidNinjaPlatform struct {
	toolchains toolchain.ToolchainSet
	env        *config.EnvironmentVariables
}

func NewAndroidNinjaPlatform(env *config.EnvironmentVariables, cfg *config.Properties) Platform {
	p := AndroidNinjaPlatform{
		env: env,
	}

	p.Init(cfg)

	return &p

}

// BinaryOutputDir implements Platform.
func (*AndroidNinjaPlatform) BinaryOutputDir(tgt toolchain.TgtType) string {
	return filepath.Join("$BuildDir", string(tgt), "executable")
}

// BobScriptsDir implements Platform.
func (*AndroidNinjaPlatform) BobScriptsDir() string {
	return "${BobScriptsDir}"
}

// BuildDir implements Platform.
func (*AndroidNinjaPlatform) BuildDir() string {
	return "${BuildDir}"
}

// EscapeFlag implements Platform.
func (*AndroidNinjaPlatform) EscapeFlag(string) string {
	panic("unimplemented")
}

// GetToolchain implements Platform.
func (g *AndroidNinjaPlatform) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	return g.toolchains.GetToolchain(tgt)
}

// Init implements Platform.
func (g *AndroidNinjaPlatform) Init(config *config.Properties) {
	g.toolchains.Configure(config)
}

// KernelModOutputDir implements Platform.
func (*AndroidNinjaPlatform) KernelModOutputDir() string {
	panic("unimplemented")
}

// SharedLibsDir implements Platform.
func (*AndroidNinjaPlatform) SharedLibsDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "shared")
}

// SourceDir implements Platform.
func (*AndroidNinjaPlatform) SourceDir() string {
	return "${SrcDir}"
}

// SourceOutputDir implements Platform.
func (*AndroidNinjaPlatform) SourceOutputDir(m blueprint.Module) string {
	return filepath.Join("${BuildDir}", "gen", m.Name())
}

// StaticLibOutputDir implements Platform.
func (*AndroidNinjaPlatform) StaticLibOutputDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "static")
}

var _ Platform = (*AndroidNinjaPlatform)(nil)
