package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/internal/utils"
)

type toolchainGnu interface {
	Toolchain
	getBinDirs() []string
	getStdCxxHeaderDirs() []string
	getInstallDir() string
}

type toolchainGnuCommon struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	objdumpBinary string
	gccBinary     string
	gxxBinary     string
	linker        Linker
	prefix        string
	cflags        []string // Flags for both C and C++
	ldflags       []string // Linker flags, including anything required for C++
	binDir        string
	flagCache     *flagSupportedCache
	is64BitOnly   bool
}

type toolchainGnuNative struct {
	toolchainGnuCommon
}

type toolchainGnuCross struct {
	toolchainGnuCommon
}

func (tc toolchainGnuCommon) GetArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainGnuCommon) GetAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainGnuCommon) GetCCompiler() (string, []string) {
	return tc.gccBinary, tc.cflags
}

func (tc toolchainGnuCommon) GetCXXCompiler() (tool string, flags []string) {
	return tc.gxxBinary, tc.cflags
}

func (tc toolchainGnuCommon) GetLinker() Linker {
	return tc.linker
}

func (tc toolchainGnuCommon) GetStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainGnuCommon) GetLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainGnuCommon) getBinDirs() []string {
	return []string{tc.binDir}
}

func (tc toolchainGnuCommon) CheckFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

// The libstdc++ headers shipped with GCC toolchains are stored, relative to
// the `prefix-gcc` binary's location, in `../$ARCH/include/c++/$VERSION` and
// `../$ARCH/include/c++/$VERSION/$ARCH`. This function returns $ARCH. This is
// generally the same as the compiler prefix, but because the prefix can
// contain the path to the compiler as well, we instead obtain it by trying the
// `-print-multiarch` and `-dumpmachine` options.
func (tc toolchainGnuCommon) getTargetTripleHeaderSubdir() string {
	ccBinary, flags := tc.GetCCompiler()
	cmd := exec.Command(ccBinary, utils.NewStringSlice(flags, []string{"-print-multiarch"})...)
	bytes, err := cmd.Output()
	if err == nil {
		target := strings.TrimSpace(string(bytes))
		if len(target) > 0 {
			return target
		}
	}

	// Some toolchains will output nothing for -print-multiarch, so try
	// -dumpmachine if it didn't work (-dumpmachine works for most
	// toolchains, but will ignore options like '-m32').
	cmd = exec.Command(ccBinary, utils.NewStringSlice(flags, []string{"-dumpmachine"})...)
	bytes, err = cmd.Output()
	if err != nil {
		panic(fmt.Errorf("Couldn't get arch directory for compiler %s: %v", ccBinary, err))
	}
	return strings.TrimSpace(string(bytes))
}

func (tc toolchainGnuCommon) getVersion() string {
	ccBinary, _ := tc.GetCCompiler()
	cmd := exec.Command(ccBinary, "-dumpversion")
	bytes, err := cmd.Output()
	if err != nil {
		panic(fmt.Errorf("Couldn't get version for compiler %s: %v", ccBinary, err))
	}
	return strings.TrimSpace(string(bytes))
}

func (tc toolchainGnuCommon) getInstallDir() string {
	return filepath.Dir(tc.binDir)
}

func (tc toolchainGnuCommon) Is64BitOnly() bool {
	return tc.is64BitOnly
}

// Prefixed standalone toolchains (e.g. aarch64-linux-gnu-gcc) often ship with a
// directory of symlinks containing un-prefixed names e.g. just 'ld', instead of
// 'aarch64-linux-gnu-ld'. Some Clang installations won't use the prefix, even
// when passed the --gcc-toolchain option, so add the unprefixed version to the
// binary search path.
func (tc toolchainGnuCross) getBinDirs() []string {
	dirs := tc.toolchainGnuCommon.getBinDirs()
	triple := tc.getTargetTripleHeaderSubdir()

	unprefixedBinDir := filepath.Join(tc.getInstallDir(), triple, "bin")
	if fi, err := os.Stat(unprefixedBinDir); !os.IsNotExist(err) && fi.IsDir() {
		dirs = append(dirs, unprefixedBinDir)
	}
	return dirs
}

func (tc toolchainGnuNative) getStdCxxHeaderDirs() []string {
	installDir := tc.getInstallDir()
	triple := tc.getTargetTripleHeaderSubdir()
	return []string{
		filepath.Join(installDir, "include", "c++", tc.getVersion()),
		filepath.Join(installDir, "include", "c++", tc.getVersion(), triple),
	}
}

func (tc toolchainGnuCross) getStdCxxHeaderDirs() []string {
	installDir := tc.getInstallDir()
	triple := tc.getTargetTripleHeaderSubdir()
	return []string{
		filepath.Join(installDir, triple, "include", "c++", tc.getVersion()),
		filepath.Join(installDir, triple, "include", "c++", tc.getVersion(), triple),
	}
}

func newToolchainGnuCommon(props *config.Properties, tgt TgtType) (tc toolchainGnuCommon) {
	tc.prefix = props.GetString(string(tgt) + "_gnu_prefix")
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")

	tc.gccBinary = tc.prefix + props.GetString(string(tgt)+"_gnu_cc_binary")
	tc.gxxBinary = tc.prefix + props.GetString(string(tgt)+"_gnu_cxx_binary")
	tc.binDir = filepath.Dir(getToolPath(tc.gccBinary))

	sysroot := props.GetString(string(tgt) + "_sysroot")
	if sysroot != "" {
		tc.cflags = append(tc.cflags, "--sysroot="+sysroot)
		tc.ldflags = append(tc.ldflags, "--sysroot="+sysroot)
	}

	gnuFlagsProp := props.GetString(string(tgt) + "_gnu_flags")
	if gnuFlagsProp != "" {
		flags := strings.Split(gnuFlagsProp, " ")
		tc.cflags = append(tc.cflags, flags...)
		tc.ldflags = append(tc.ldflags, flags...)
	}

	tc.linker = newDefaultLinker(tc.gxxBinary, tc.ldflags, []string{})
	tc.flagCache = newFlagCache()
	tc.is64BitOnly = props.GetBool(string(tgt) + "_64bit_only")

	return
}

func newToolchainGnuNative(props *config.Properties) (tc toolchainGnuNative) {
	tc.toolchainGnuCommon = newToolchainGnuCommon(props, TgtTypeHost)
	return
}

func newToolchainGnuCross(props *config.Properties) (tc toolchainGnuCross) {
	tc.toolchainGnuCommon = newToolchainGnuCommon(props, TgtTypeTarget)
	return
}
