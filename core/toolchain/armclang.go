package toolchain

import (
	"strings"

	"github.com/ARM-software/bob-build/core/config"
)

type toolchainArmClang struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	objdumpBinary string
	ccBinary      string
	cxxBinary     string
	linker        Linker
	prefix        string
	cflags        []string // Flags for both C and C++
	flagCache     *flagSupportedCache

	is64BitOnly bool
}

type toolchainArmClangNative struct {
	toolchainArmClang
}

type toolchainArmClangCross struct {
	toolchainArmClang
}

func (tc toolchainArmClang) GetArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainArmClang) GetAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainArmClang) GetCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainArmClang) GetCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

func (tc toolchainArmClang) GetLinker() Linker {
	return tc.linker
}

func (tc toolchainArmClang) GetStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainArmClang) GetLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainArmClang) CheckFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func (tc toolchainArmClang) Is64BitOnly() bool {
	return tc.is64BitOnly
}

func newToolchainArmClangCommon(props *config.Properties, tgt TgtType) (tc toolchainArmClang) {
	tc.prefix = props.GetString(string(tgt) + "_gnu_prefix")
	tc.arBinary = tc.prefix + props.GetString("armclang_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("armclang_as_binary")
	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")
	tc.ccBinary = tc.prefix + props.GetString(string(tgt)+"_armclang_cc_binary")
	tc.cxxBinary = tc.prefix + props.GetString(string(tgt)+"_armclang_cxx_binary")
	tc.linker = newDefaultLinker(tc.cxxBinary, []string{}, []string{})

	tc.cflags = strings.Split(props.GetString(string(tgt)+"_armclang_flags"), " ")
	tc.flagCache = newFlagCache()
	tc.is64BitOnly = props.GetBool(string(tgt) + "_64bit_only")

	return
}

func newToolchainArmClangNative(props *config.Properties) (tc toolchainArmClangNative) {
	tc.toolchainArmClang = newToolchainArmClangCommon(props, TgtTypeHost)
	return
}

func newToolchainArmClangCross(props *config.Properties) (tc toolchainArmClangCross) {
	tc.toolchainArmClang = newToolchainArmClangCommon(props, TgtTypeTarget)
	return
}
