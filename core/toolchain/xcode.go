package toolchain

import "github.com/ARM-software/bob-build/core/config"

type toolchainXcode struct {
	arBinary    string
	asBinary    string
	dsymBinary  string
	stripBinary string
	otoolBinary string
	nmBinary    string
	ccBinary    string
	cxxBinary   string
	linker      Linker
	prefix      string
	target      string
	flagCache   *flagSupportedCache

	cflags  []string
	ldflags []string

	is64BitOnly bool
}

type toolchainXcodeNative struct {
	toolchainXcode
}

type toolchainXcodeCross struct {
	toolchainXcode
}

func (tc toolchainXcode) GetArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainXcode) GetAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainXcode) GetCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainXcode) GetCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

func (tc toolchainXcode) GetLinker() Linker {
	return tc.linker
}

func (tc toolchainXcode) GetStripFlags() []string {
	return []string{
		"--format", "macho",
		"--dsymutil-tool", tc.dsymBinary,
		"--strip-tool", tc.stripBinary,
	}
}

func (tc toolchainXcode) GetLibraryTocFlags() []string {
	return []string{
		"--format", "macho",
		"--otool-tool", tc.otoolBinary,
		"--nm-tool", tc.nmBinary,
	}
}

func (tc toolchainXcode) CheckFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func (tc toolchainXcode) Is64BitOnly() bool {
	return tc.is64BitOnly
}

func newToolchainXcodeCommon(props *config.Properties, tgt TgtType) (tc toolchainXcode) {
	tc.prefix = props.GetString(string(tgt) + "_xcode_prefix")
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")
	tc.dsymBinary = props.GetString(string(tgt) + "_dsymutil_binary")
	tc.stripBinary = props.GetString(string(tgt) + "_strip_binary")
	tc.otoolBinary = props.GetString(string(tgt) + "_otool_binary")
	tc.nmBinary = props.GetString(string(tgt) + "_nm_binary")

	tc.ccBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cc_binary")
	tc.cxxBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cxx_binary")

	tc.target = props.GetString(string(tgt) + "_xcode_triple")

	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
		tc.ldflags = append(tc.ldflags, "-target", tc.target)
	}

	tc.linker = newXcodeLinker(tc.cxxBinary, tc.ldflags, []string{})
	tc.flagCache = newFlagCache()
	tc.is64BitOnly = props.GetBool(string(tgt) + "_64bit_only")
	return
}

func newToolchainXcodeNative(props *config.Properties) (tc toolchainXcodeNative) {
	tc.toolchainXcode = newToolchainXcodeCommon(props, TgtTypeHost)
	return
}

func newToolchainXcodeCross(props *config.Properties) (tc toolchainXcodeCross) {
	tc.toolchainXcode = newToolchainXcodeCommon(props, TgtTypeTarget)
	return
}
