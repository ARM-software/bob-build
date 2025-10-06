package toolchain

import (
	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/internal/utils"
)

type toolchainClangCommon struct {
	// Options read from the config:
	arBinary       string
	asBinary       string
	objcopyBinary  string
	objdumpBinary  string
	clangBinary    string
	clangxxBinary  string
	linker         Linker
	prefix         string
	useGnuBinutils bool

	// Use the GNU Toolchain's 'ar' and 'as', as well as its libstdc++
	// headers if required
	gnu toolchainGnu

	// Calculated during Toolchain initialization:
	cflags   []string // Flags for both C and C++
	cxxflags []string // Flags just for C++
	ldflags  []string // Linker flags, including anything required for C++
	ldlibs   []string // Linker libraries

	target    string
	flagCache *flagSupportedCache

	is64BitOnly bool
}

type toolchainClangNative struct {
	toolchainClangCommon
}

type toolchainClangCross struct {
	toolchainClangCommon
}

func (tc toolchainClangCommon) GetArchiver() (string, []string) {
	if tc.useGnuBinutils {
		return tc.gnu.GetArchiver()
	}
	return tc.arBinary, []string{}
}

func (tc toolchainClangCommon) GetAssembler() (string, []string) {
	if tc.useGnuBinutils {
		return tc.gnu.GetAssembler()
	}
	return tc.asBinary, []string{}
}

func (tc toolchainClangCommon) GetCCompiler() (string, []string) {
	return tc.clangBinary, tc.cflags
}

func (tc toolchainClangCommon) GetCXXCompiler() (string, []string) {
	return tc.clangxxBinary, tc.cxxflags
}

func (tc toolchainClangCommon) GetLinker() Linker {
	return newDefaultLinker(tc.clangxxBinary, tc.ldflags, tc.ldlibs)
}

func (tc toolchainClangCommon) GetStripFlags() []string {
	return []string{
		"--format", "elf",
		"--objcopy-tool", tc.objcopyBinary,
	}
}

func (tc toolchainClangCommon) GetLibraryTocFlags() []string {
	return []string{
		"--format", "elf",
		"--objdump-tool", tc.objdumpBinary,
	}
}

func (tc toolchainClangCommon) CheckFlagIsSupported(language, flag string) bool {
	return tc.flagCache.checkFlag(tc, language, flag)
}

func (tc toolchainClangCommon) Is64BitOnly() bool {
	return tc.is64BitOnly
}

func newToolchainClangCommon(props *config.Properties, tgt TgtType) (tc toolchainClangCommon) {
	tc.prefix = props.GetString(string(tgt) + "_clang_prefix")

	// This assumes arBinary and asBinary are either in the path, or the same directory as clang.
	// This is not necessarily the case. This will need to be updated when we support clang on linux without a GNU Toolchain.
	tc.arBinary = props.GetString(string(tgt) + "_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.objdumpBinary = props.GetString(string(tgt) + "_objdump_binary")

	tc.clangBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cc_binary")
	tc.clangxxBinary = tc.prefix + props.GetString(string(tgt)+"_clang_cxx_binary")

	tc.target = props.GetString(string(tgt) + "_clang_triple")

	// Here we add flags relating to android out of tree builds
	if out_of_tree := props.GetBool("builder_android_ninja"); out_of_tree {
		tc.cflags = append(tc.cflags, "-DANDROID")
		tc.cflags = append(tc.cflags, "-Wno-unused-but-set-variable")

		if tgt == TgtTypeTarget {
			tc.target = "aarch64-linux-android10000" // TODO: This should detect API version instead of dev?
			tc.cflags = append(tc.cflags, "-march=armv8-a+crypto+sha2")
			tc.cflags = append(tc.cflags, "-nostdlibinc")
			tc.cflags = append(tc.cflags, "-fPIC")
			tc.cflags = append(tc.cflags, "-Wno-nullability-extension", "-Wno-gcc-compat")
			tc.cflags = append(tc.cflags, "-Wno-deprecated-non-prototypes", "-Wno-shorten-64-to-32", "-Wno-unused-but-set-variable")
			tc.cflags = append(tc.cflags, "-Wno-implicit-function-declaration", "-Wno-int-conversion")
			android_build_top := props.GetString("android_build_top")
			tc.cflags = append(tc.cflags, "-I"+android_build_top+"/prebuilts/clang/host/linux-x86/clang-r522817/include/c++/v1/",
				"-isystem "+android_build_top+"/prebuilts/vndk/v34/arm64/include/generated-headers/bionic/libc/libc/android_vendor.34_arm64_armv8-a_shared/gen/include",
				"-isystem "+android_build_top+"/prebuilts/vndk/v34/arm64/include/bionic/libc/kernel/uapi/asm-arm64/",
				"-isystem "+android_build_top+"/prebuilts/vndk/v34/arm64/include/bionic/libc/kernel/android/uapi/",
				"-isystem "+android_build_top+"/prebuilts/vndk/v34/arm64/include/bionic/libc/kernel/uapi/",
				"-isystem "+android_build_top+"/prebuilts/runtime/mainline/runtime/sdk/common_os/include/bionic/libc",
			)

			tc.cflags = append(tc.cflags,
				"-Wno-format-insufficient-args",
				"-Wno-misleading-indentation",
				"-Wno-bitwise-instead-of-logical",
				"-Wno-unused",
				"-Wno-unused-parameter",
				"-Wno-unused-but-set-parameter",
				"-Wno-unqualified-std-cast-call",
				"-Wno-array-parameter",
				"-Wno-gnu-offsetof-extensions",
				"-Wno-fortify-source",
				"-Wno-tautological-constant-compare",
				"-Wno-tautological-type-limit-compare",
				"-Wno-implicit-int-float-conversion",
				"-Wno-tautological-overlap-compare",
				"-Wno-deprecated-copy",
				"-Wno-range-loop-construct",
				"-Wno-zero-as-null-pointer-constant",
				"-Wno-deprecated-anon-enum-enum-conversion",
				"-Wno-deprecated-enum-enum-conversion",
				"-Wno-pessimizing-move",
				"-Wno-non-c-typedef-for-linkage",
				"-Wno-align-mismatch",
				"-Wno-error=unused-but-set-variable",
				"-Wno-error=unused-but-set-parameter",
				"-Wno-error=deprecated-builtins",
				"-Wno-error=deprecated",
				"-Wno-error=single-bit-bitfield-constant-conversion",
				"-Wno-error=enum-constexpr-conversion",
				"-Wno-error=invalid-offsetof",
				"-Wno-error=thread-safety-reference-return",
				"-Wno-deprecated-dynamic-exception-spec",
				"-Wno-vla-cxx-extension",
				"-Wno-unused-variable",
				"-Wno-missing-field-initializers",
				"-Wno-packed-non-pod",
				"-Wno-void-pointer-to-enum-cast",
				"-Wno-void-pointer-to-int-cast",
				"-Wno-pointer-to-int-cast",
				"-Wno-error=deprecated-declarations",
				"-Wno-missing-field-initializers",
				"-Wno-gnu-include-next",
				"-Wno-unused-function",
				"-Wno-missing-field-initializers",
				"-Wno-unused-parameter",
				"-Wno-tautological-constant-out-of-range-compare",
				"-Wno-unknown-warning-option",
				"-Wno-tautological-constant-out-of-range-compare",
				"-Wno-duplicate-decl-specifier",
				"-Wno-format-pedantic",
				"-Wno-gnu-zero-variadic-macro-arguments",
				"-Wno-gnu-redeclared-enum",
				"-Wno-newline-eof",
				"-Wno-expansion-to-defined",
				"-Wno-embedded-directive",
				"-Wno-implicit-fallthrough",
				"-Wno-zero-length-array",
				"-Wno-c11-extensions",
				"-Wno-gnu-include-next",
				"-Wno-long-long",
				"-Wno-variadic-macros",
				"-Wno-overlength-strings",
				"-Wno-attributes",
				"-Wno-unused-parameter",
				"-Wno-type-limits",
				"-Wno-error=nested-anon-types",
				"-Wno-error=gnu-anonymous-struct",
				"-Wno-missing-field-initializers",
				"-Wno-disabled-macro-expansion",
				"-Wno-padded",
				"-Wno-unused-macros",
				"-Wno-c++98-compat",
				"-Wno-c++98-compat-pedantic",
				"-Wno-c++2a-compat",
				"-Wno-c++2a-compat-pedantic",
				"-Wno-return-std-move-in-c++11",
				"-Wno-reserved-identifier",
				"-Wno-gnu-zero-variadic-macro-arguments",
				"-Wno-enum-compare",
				"-Wno-enum-compare-switch",
				"-Wno-null-pointer-arithmetic",
				"-Wno-null-dereference",
				"-Wno-pointer-compare",
				"-Wno-final-dtor-non-final-class",
				"-Wno-psabi",
				"-Wno-null-pointer-subtraction",
				"-Wno-string-concatenation",
				"-Wno-deprecated-non-prototype",
				"-Wno-unused",
				"-Wno-deprecated",
				"-Wno-error=deprecated-declarations",
				"-Wno-c99-designator",
				"-Wno-gnu-folding-constant",
				"-Wno-inconsistent-missing-override",
				"-Wno-error=reorder-init-list",
				"-Wno-reorder-init-list",
				"-Wno-sign-compare",
				"-Wno-unused",
				"-Wno-strict-prototypes",
			)
		} else { // tgtType == Host
			android_build_top := props.GetString("android_build_top")
			// TODO: Make the clang revision settable from Mconfig
			tc.cflags = append(tc.cflags,
				"-I"+android_build_top+"/prebuilts/clang/host/linux-x86/clang-r522817/include/x86_64-unknown-linux-gnu/c++/v1/",
				"-I"+android_build_top+"/prebuilts/clang/host/linux-x86/clang-r522817/include/c++/v1/")

			tc.cflags = append(tc.cflags,
				"-Wno-deprecated-enum-enum-conversion",
				"-Wno-pessimizing-move",
				"-Wno-non-c-typedef-for-linkage",
				"-Wno-align-mismatch",
				"-Wno-error=unused-but-set-variable",
				"-Wno-error=unused-but-set-parameter",
				"-Wno-error=deprecated-builtins",
				"-Wno-error=deprecated",
				"-Wno-error=single-bit-bitfield-constant-conversion",
				"-Wno-error=enum-constexpr-conversion",
				"-Wno-error=invalid-offsetof",
				"-Wno-error=thread-safety-reference-return",
				"-Wno-deprecated-dynamic-exception-spec",
				"-Wno-vla-cxx-extension",
				"-fcommon",
				"-Wno-format-insufficient-args",
				"-Wno-misleading-indentation",
				"-Wno-bitwise-instead-of-logical",
				"-Wno-unused",
				"-Wno-unused-parameter",
				"-Wno-unused-but-set-parameter",
				"-Wno-unqualified-std-cast-call",
				"-Wno-array-parameter",
				"-Wno-gnu-offsetof-extensions",
				"-Wno-fortify-source",
				"-D__STDC_CONSTANT_MACROS",
				"-D__STDC_LIMIT_MACROS",
				"-fvisibility-inlines-hidden",
				"-fno-exceptions",
				"-Wno-error=deprecated-declarations",
				"-fexceptions",
				"-Wno-shadow",
				"-D_GNU_SOURCE=1",
				"-ffunction-sections",
				"-fdata-sections",
				"-Qunused-arguments",
				"-fcolor-diagnostics",
				"-fno-exceptions",
				"-fno-unwind-tables",
				"-pedantic",
				"-Wno-long-long",
				"-Wno-variadic-macros",
				"-Wno-overlength-strings",
				"-Wno-attributes",
				"-Wno-unused-parameter",
				"-Wno-type-limits",
				"-Wno-missing-field-initializers",
				"-Wno-unknown-warning-option",
				"-Wno-tautological-constant-out-of-range-compare",
				"-Wno-duplicate-decl-specifier",
				"-Wno-extended-offsetof",
				"-Wno-format-pedantic",
				"-Wno-gnu-zero-variadic-macro-arguments",
				"-Wno-gnu-redeclared-enum",
				"-Wno-newline-eof",
				"-Wno-expansion-to-defined",
				"-Wno-embedded-directive",
				"-Wno-implicit-fallthrough",
				"-Wno-zero-length-array",
				"-Wno-c11-extensions",
				"-Wno-gnu-include-next",
				"-DCFRAMEP_DUMP=0",
				"-Wno-unused-function",
				"-Wno-missing-field-initializers",
				"-Wno-unused-parameter",
				"-Wno-tautological-constant-out-of-range-compare",
			)

			tc.cflags = append(tc.cflags,
				"-Wno-format-insufficient-args",
				"-Wno-misleading-indentation",
				"-Wno-bitwise-instead-of-logical",
				"-Wno-unused",
				"-Wno-unused-parameter",
				"-Wno-unused-but-set-parameter",
				"-Wno-unqualified-std-cast-call",
				"-Wno-array-parameter",
				"-Wno-gnu-offsetof-extensions",
				"-Wno-fortify-source",
				"-Wno-tautological-constant-compare",
				"-Wno-tautological-type-limit-compare",
				"-Wno-implicit-int-float-conversion",
				"-Wno-tautological-overlap-compare",
				"-Wno-deprecated-copy",
				"-Wno-range-loop-construct",
				"-Wno-zero-as-null-pointer-constant",
				"-Wno-deprecated-anon-enum-enum-conversion",
				"-Wno-deprecated-enum-enum-conversion",
				"-Wno-pessimizing-move",
				"-Wno-non-c-typedef-for-linkage",
				"-Wno-align-mismatch",
				"-Wno-error=unused-but-set-variable",
				"-Wno-error=unused-but-set-parameter",
				"-Wno-error=deprecated-builtins",
				"-Wno-error=deprecated",
				"-Wno-error=single-bit-bitfield-constant-conversion",
				"-Wno-error=enum-constexpr-conversion",
				"-Wno-error=invalid-offsetof",
				"-Wno-error=thread-safety-reference-return",
				"-Wno-deprecated-dynamic-exception-spec",
				"-Wno-vla-cxx-extension",
				"-Wno-unused-variable",
				"-Wno-missing-field-initializers",
				"-Wno-packed-non-pod",
				"-Wno-void-pointer-to-enum-cast",
				"-Wno-void-pointer-to-int-cast",
				"-Wno-pointer-to-int-cast",
				"-Wno-error=deprecated-declarations",
				"-Wno-missing-field-initializers",
				"-Wno-gnu-include-next",
				"-Wno-unused-function",
				"-Wno-missing-field-initializers",
				"-Wno-unused-parameter",
				"-Wno-tautological-constant-out-of-range-compare",
				"-Wno-unknown-warning-option",
				"-Wno-tautological-constant-out-of-range-compare",
				"-Wno-duplicate-decl-specifier",
				"-Wno-format-pedantic",
				"-Wno-gnu-zero-variadic-macro-arguments",
				"-Wno-gnu-redeclared-enum",
				"-Wno-newline-eof",
				"-Wno-expansion-to-defined",
				"-Wno-embedded-directive",
				"-Wno-implicit-fallthrough",
				"-Wno-zero-length-array",
				"-Wno-c11-extensions",
				"-Wno-gnu-include-next",
				"-Wno-long-long",
				"-Wno-variadic-macros",
				"-Wno-overlength-strings",
				"-Wno-attributes",
				"-Wno-unused-parameter",
				"-Wno-type-limits",
				"-Wno-error=nested-anon-types",
				"-Wno-error=gnu-anonymous-struct",
				"-Wno-missing-field-initializers",
				"-Wno-disabled-macro-expansion",
				"-Wno-padded",
				"-Wno-unused-macros",
				"-Wno-c++98-compat",
				"-Wno-c++98-compat-pedantic",
				"-Wno-c++2a-compat",
				"-Wno-c++2a-compat-pedantic",
				"-Wno-return-std-move-in-c++11",
				"-Wno-reserved-identifier",
				"-Wno-gnu-zero-variadic-macro-arguments",
				"-Wno-enum-compare",
				"-Wno-enum-compare-switch",
				"-Wno-null-pointer-arithmetic",
				"-Wno-null-dereference",
				"-Wno-pointer-compare",
				"-Wno-final-dtor-non-final-class",
				"-Wno-psabi",
				"-Wno-null-pointer-subtraction",
				"-Wno-string-concatenation",
				"-Wno-deprecated-non-prototype",
				"-Wno-unused",
				"-Wno-deprecated",
				"-Wno-error=deprecated-declarations",
				"-Wno-c99-designator",
				"-Wno-gnu-folding-constant",
				"-Wno-inconsistent-missing-override",
				"-Wno-error=reorder-init-list",
				"-Wno-reorder-init-list",
				"-Wno-sign-compare",
				"-Wno-unused",
				"-Wno-strict-prototypes",
				"-Wno-macro-redefined",
				"-lrt",
			)

			tc.cflags = append(tc.cflags,
				"-target x86_64-linux-gnu",
				"-nostdlib++",
				"-m64",
				"-lc++",
				"-L"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/lib/gcc/x86_64-linux/4.8.3/",
				"-L"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/x86_64-linux/lib64/",
				"-B"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/lib/gcc/x86_64-linux/4.8.3/",
				"--sysroot="+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot",
				"-fuse-ld=lld",
				"-Wl,--icf=safe",
				"-Wl,--no-demangle",
				"-Wa,--noexecstack",
				"-fPIC -U_FORTIFY_SOURCE -D_FORTIFY_SOURCE=2 -fstack-protector",
				"--gcc-toolchain="+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/",
				"-fstack-protector-strong",
			)

			tc.ldflags = append(tc.cflags,
				"-target x86_64-linux-gnu",
				"-nostdlib++",
				"-m64",
				"-lc++",
				"-L"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/lib/gcc/x86_64-linux/4.8.3/",
				"-L"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/x86_64-linux/lib64/",
				"-B"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/lib/gcc/x86_64-linux/4.8.3/",
				"--sysroot="+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/sysroot",
				"-fuse-ld=lld",
				"-Wl,--icf=safe",
				"-Wl,--no-demangle",
				"-Wa,--noexecstack",
				"-fPIC -U_FORTIFY_SOURCE -D_FORTIFY_SOURCE=2 -fstack-protector",
				"--gcc-toolchain="+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/",
				"-fstack-protector-strong",
				android_build_top+"/prebuilts/clang/host/linux-x86/clang-r522817/lib/x86_64-unknown-linux-gnu/libc++.so",
				// TODO: Relative rpaths
				"-Wl,-rpath,"+android_build_top+"/prebuilts/gcc/linux-x86/host/x86_64-linux-glibc2.17-4.8/x86_64-linux/lib64/",
				"-Wl,-rpath,"+android_build_top+"/prebuilts/build-tools/linux-x86/lib64/", // this needs to be exported
			)
		}
	}

	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
		tc.ldflags = append(tc.ldflags, "-target", tc.target)
	}

	// TODO Mirror the platform flag code to the GCC toolchain
	// TODO Make a separate bazel toolchain

	if cxxflags := props.GetStringIfExists(string(tgt) + "_cxxflags"); cxxflags != "" {
		tc.cxxflags = append(tc.cxxflags, cxxflags)
	}

	if ldflags := props.GetStringIfExists(string(tgt) + "_ldflags"); ldflags != "" {
		tc.ldflags = append(tc.ldflags, ldflags)
	}

	sysroot := props.GetString(string(tgt) + "_sysroot")
	if sysroot != "" {
		tc.cflags = append(tc.cflags, "--sysroot="+sysroot)
		tc.ldflags = append(tc.ldflags, "--sysroot="+sysroot)
	}

	stl := props.GetString(string(tgt) + "_clang_stl_library")
	rt := props.GetString(string(tgt) + "_clang_compiler_runtime")
	useGnuCrt := props.GetBool(string(tgt) + "_clang_use_gnu_crt")
	useGnuStl := props.GetBool(string(tgt) + "_clang_use_gnu_stl")
	useGnuLibgcc := props.GetBool(string(tgt) + "_clang_use_gnu_libgcc")

	tc.useGnuBinutils = props.GetBool(string(tgt) + "_clang_use_gnu_binutils")

	if tc.useGnuBinutils || useGnuStl || useGnuCrt || useGnuLibgcc {
		if tgt == TgtTypeHost {
			tc.gnu = newToolchainGnuNative(props)
		} else {
			tc.gnu = newToolchainGnuCross(props)
		}
	}

	if stl != "" {
		tc.cxxflags = append(tc.cxxflags, "--stdlib=lib"+stl)
		tc.ldflags = append(tc.ldflags, "--stdlib=lib"+stl)
	}

	if rt != "" {
		tc.cflags = append(tc.cflags, "--rtlib="+rt)
		tc.ldflags = append(tc.ldflags, "--rtlib="+rt)
	}

	binDirs := []string{}

	if useGnuCrt || useGnuLibgcc || useGnuStl {
		// Tell Clang where the GNU Toolchain is installed, so it can use its
		// headers and libraries, for example, if we are using libstdc++.
		gnuInstallArg := "--gcc-toolchain=" + tc.gnu.getInstallDir()
		tc.cflags = append(tc.cflags, gnuInstallArg)
		tc.ldflags = append(tc.ldflags, gnuInstallArg)
	}
	if useGnuCrt {
		binDirs = append(binDirs, getFileNameDir(tc.gnu, "crt1.o")...)
	}
	if tc.useGnuBinutils {
		// Add the GNU Toolchain's binary directories to Clang's binary search
		// path, so that Clang can find the correct linker. If the GNU Toolchain
		// is a "system" Toolchain (e.g. in /usr/bin), its binaries will already
		// be in Clang's search path, so these arguments have no effect.
		binDirs = append(binDirs, tc.gnu.getBinDirs()...)
	}

	tc.ldflags = append(tc.ldflags, utils.PrefixAll(binDirs, "-B")...)

	if useGnuLibgcc {
		dirs := utils.AppendUnique(getFileNameDir(tc.gnu, "libgcc.a"),
			getFileNameDir(tc.gnu, "libgcc_s.so"))
		tc.ldflags = append(tc.ldflags, utils.PrefixAll(dirs, "-L")...)
	}

	if useGnuStl {
		tc.cxxflags = append(tc.cxxflags,
			utils.PrefixAll(tc.gnu.getStdCxxHeaderDirs(), "-isystem ")...)
	}

	if rt == "libgcc" {
		// GCC __atomic__ builtins are provided by GNU libatomic.
		// Clang supports them via compiler-rt. However clang does not
		// link against libatomic automatically when libgcc is the
		// compiler runtime. libatomic is only needed for certain
		// architectures, so check whether it is present before trying
		// to link against it.
		//
		// libatomic is expected to be in the same dir as libgcc, so
		// the check of whether it is present must happen after adding
		// the -L for libgcc (if needed). We expect an error.
		_, err := getFileName(tc, "libatomic.so")
		if err != nil {
			tc.ldlibs = append(tc.ldlibs, "-latomic")
		}
	}

	// Combine cflags and cxxflags once here, to avoid appending during
	// every call to GetCXXCompiler().
	tc.cxxflags = append(tc.cxxflags, tc.cflags...)

	if cflags := props.GetStringIfExists(string(tgt) + "_cflags"); cflags != "" {
		tc.cflags = append(tc.cflags, cflags)
	}

	tc.linker = newDefaultLinker(tc.clangxxBinary, tc.cflags, []string{})
	tc.flagCache = newFlagCache()
	tc.is64BitOnly = props.GetBool(string(tgt) + "_64bit_only")

	return
}

func newToolchainClangNative(props *config.Properties) (tc toolchainClangNative) {
	tc.toolchainClangCommon = newToolchainClangCommon(props, TgtTypeHost)
	return
}

func newToolchainClangCross(props *config.Properties) (tc toolchainClangCross) {
	tc.toolchainClangCommon = newToolchainClangCommon(props, TgtTypeTarget)
	return
}
