// The backend package provides platform specific configuration for the generator.
// Based on the type of generator used, different paths will be required.
package backend

import (
	"sync"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/ARM-software/bob-build/internal/warnings"
	"github.com/google/blueprint"
)

// Backend platform singleton.
// Abstracts platform specific (AOSP, Ninja) parameters and stores toolchains.
type Platform interface {
	BuildDir() string
	SourceDir() string
	BobScriptsDir() string
	SourceOutputDir(blueprint.Module) string
	SharedLibsDir(tgt toolchain.TgtType) string
	StaticLibOutputDir(tgt toolchain.TgtType) string
	BinaryOutputDir(tgt toolchain.TgtType) string
	KernelModOutputDir() string
	EscapeFlag(string) string
	Init(*config.Properties)
	GetToolchain(tgt toolchain.TgtType) toolchain.Toolchain
	GetLogger() *warnings.WarningLogger
}

var platform Platform
var lock = &sync.Mutex{}

func Get() Platform {
	if platform == nil {
		lock.Lock()
		defer lock.Unlock()
		if platform == nil {
			panic("The Backend Platform has not been configured, please call Setup() before use.")
		}
	}

	return platform
}

func Setup(env *config.EnvironmentVariables, cfg *config.Properties, logger *warnings.WarningLogger) {
	if platform == nil {
		lock.Lock()
		defer lock.Unlock()
		if platform == nil {
			switch {
			case cfg.GetBool("builder_ninja"):
				platform = NewLinuxPlatform(env, cfg, logger)
			case cfg.GetBool("builder_android_bp"):
				platform = NewAndroidPlatform(env, cfg, logger)
			default:
				utils.Die("Unknown builder backend")
			}
		}
	}
}
