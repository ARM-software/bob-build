package backend

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type LinuxPlatform struct {
	toolchains toolchain.ToolchainSet
}

var _ Platform = (*LinuxPlatform)(nil)

func (g *LinuxPlatform) BuildDir() string {
	return "${BuildDir}"
}

func (g *LinuxPlatform) SourceDir() string {
	return "${SrcDir}"
}

func (g *LinuxPlatform) BobScriptsDir() string {
	return "${BobScriptsDir}"
}

func (g *LinuxPlatform) SourceOutputDir(m blueprint.Module) string {
	return filepath.Join("${BuildDir}", "gen", m.Name())
}

func (g *LinuxPlatform) SharedLibsDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "shared")
}

func (g *LinuxPlatform) StaticLibOutputDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "static")
}

func (g *LinuxPlatform) BinaryOutputDir(tgt toolchain.TgtType) string {
	return filepath.Join("${BuildDir}", string(tgt), "executable")
}

func (g *LinuxPlatform) KernelModOutputDir() string {
	return filepath.Join("${BuildDir}", "target", "kernel_modules")
}

func (g *LinuxPlatform) EscapeFlag(s string) string {
	return proptools.NinjaAndShellEscape(s)
}

func (g *LinuxPlatform) Init(config *config.Properties) {
	g.toolchains.Configure(config)
}

func (g *LinuxPlatform) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	return g.toolchains.GetToolchain(tgt)
}

func NewLinuxPlatform(_ *config.EnvironmentVariables, cfg *config.Properties) Platform {
	l := LinuxPlatform{}

	l.Init(cfg)

	return &l
}
