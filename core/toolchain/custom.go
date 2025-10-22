package toolchain

import (
	"github.com/ARM-software/bob-build/core/config"
)

type toolchainCustom struct {
	// Options read from the config:
	arBinary      string
	asBinary      string
	objcopyBinary string
	objdumpBinary string
	ccBinary      string // C Compiler
	cxxBinary     string // C++ Compiler
	linker        Linker

	// Calculated during Toolchain initialization:
	cflags   []string // Flags for both C and C++
	cxxflags []string // Flags just for C++
	ldflags  []string // Linker flags, including anything required for C++
	ldlibs   []string // Linker libraries

	flagCache *flagSupportedCache

	is64BitOnly bool
}

func (tc toolchainCustom) GetArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainCustom) GetAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainCustom) GetCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainCustom) GetCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cxxflags
}

func (tc toolchainCustom) GetLinker() Linker {
	return newDefaultLinker(tc.cxxBinary, tc.ldflags, tc.ldlibs)
}

func (tc toolchainCustom) GetStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainCustom) GetLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainCustom) CheckFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func (tc toolchainCustom) Is64BitOnly() bool {
	return tc.is64BitOnly
}

func newToolchainCustom(props *config.Properties, tgt TgtType) (tc toolchainCustom) {
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")

	tc.ccBinary = props.GetString(string(tgt) + "_cc_binary")
	tc.cxxBinary = props.GetString(string(tgt) + "_cxx_binary")

	if cxxflags := props.GetStringIfExists(string(tgt) + "_cxxflags"); cxxflags != "" {
		tc.cxxflags = append(tc.cxxflags, cxxflags)
	}

	if ldflags := props.GetStringIfExists(string(tgt) + "_ldflags"); ldflags != "" {
		tc.ldflags = append(tc.ldflags, ldflags)
	}

	if cflags := props.GetStringIfExists(string(tgt) + "_cflags"); cflags != "" {
		tc.cflags = append(tc.cflags, cflags)
	}

	tc.linker = newCustomLinker(tc.cxxBinary, tc.cflags, []string{})
	tc.flagCache = newFlagCache()
	tc.is64BitOnly = props.GetBool(string(tgt) + "_64bit_only")

	return
}

func newToolchainCustomNative(props *config.Properties) (tc toolchainCustom) {
	tc = newToolchainCustom(props, TgtTypeHost)
	return
}

func newToolchainCustomCross(props *config.Properties) (tc toolchainCustom) {
	tc = newToolchainCustom(props, TgtTypeTarget)
	return
}
