package backend

import (
	"path/filepath"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/google/blueprint"
)

type AndroidPlatform struct {
	toolchains toolchain.ToolchainSet
	env        *config.EnvironmentVariables
}

var _ Platform = (*AndroidPlatform)(nil)

func (g *AndroidPlatform) BuildDir() string {
	// The androidbp backend writes an Android.bp file, which should
	// never reference an actual output directory (which will be
	// chosen by Soong). Therefore this function returns an empty
	// string.
	return ""
}

func (g *AndroidPlatform) SourceDir() string {
	// On the androidbp backend, sourceDir() is only used for match_src
	// handling and for locating bob scripts. In these cases we want
	// paths relative to ANDROID_BUILD_TOP directory,
	// which is where all commands will be executed from
	return g.env.SrcDir
}

func (g *AndroidPlatform) BobScriptsDir() string {
	// In the androidbp backend, we just want the relative path to the
	// script directory.
	srcToScripts, _ := filepath.Rel(g.SourceDir(), filepath.Join(g.env.BobDir, "scripts"))
	return srcToScripts
}

func (g *AndroidPlatform) SourceOutputDir(m blueprint.Module) string {
	return ""
}

func (g *AndroidPlatform) SharedLibsDir(toolchain.TgtType) string {
	// When writing link commands, it's common to put all the shared
	// libraries in a single location to make it easy for the linker to
	// find them. This function tells us where this is for the current
	// generatorBackend.
	//
	// In the androidbp backend, we don't write link command lines. Soong
	// will do this after processing the generated Android.bp. Therefore
	// this function just returns an empty string.
	return ""
}

func (g *AndroidPlatform) StaticLibOutputDir(tgt toolchain.TgtType) string {
	return ""
}

func (g *AndroidPlatform) BinaryOutputDir(toolchain.TgtType) string {
	return ""
}

func (g *AndroidPlatform) KernelModOutputDir() string {
	return ""
}

func (g *AndroidPlatform) EscapeFlag(s string) string {
	// Soong will handle the escaping of flags, so the androidbp backend
	// just passes them through.
	return s
}

func (g *AndroidPlatform) Init(config *config.Properties) {
	g.toolchains.Configure(config)
}

func (g *AndroidPlatform) GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain {
	return g.toolchains.GetToolchain(tgt)
}

func NewAndroidPlatform(env *config.EnvironmentVariables, cfg *config.Properties) Platform {
	p := AndroidPlatform{
		env: env,
	}

	p.Init(cfg)

	return &p
}
