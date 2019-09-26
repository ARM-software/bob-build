/*
 * Copyright 2018-2019 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ARM-software/bob-build/utils"
)

type linker interface {
	getTool() string
	getFlags() []string
	getLibs() []string
	keepUnusedDependencies() string
	dropUnusedDependencies() string
	setRpathLink(string) string
	setRpath([]string) string
	linkWholeArchives([]string) string
	keepSharedLibraryTransitivity() string
	dropSharedLibraryTransitivity() string
	getForwardingLibFlags() string
}

type defaultLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l defaultLinker) getTool() string {
	return l.tool
}

func (l defaultLinker) getFlags() []string {
	return l.flags
}

func (l defaultLinker) getLibs() []string {
	return l.libs
}

func (l defaultLinker) keepUnusedDependencies() string {
	return "-Wl,--no-as-needed"
}

func (l defaultLinker) dropUnusedDependencies() string {
	return "-Wl,--as-needed"
}

func (l defaultLinker) keepSharedLibraryTransitivity() string {
	return "-Wl,--copy-dt-needed-entries"
}

func (l defaultLinker) dropSharedLibraryTransitivity() string {
	return "-Wl,--no-copy-dt-needed-entries"
}

func (l defaultLinker) getForwardingLibFlags() string {
	return "-fuse-ld=bfd"
}

func (l defaultLinker) setRpathLink(path string) string {
	return "-Wl,-rpath-link," + path
}

func (l defaultLinker) setRpath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("-Wl,--enable-new-dtags")
	for _, p := range paths {
		fmt.Fprintf(&b, ",-rpath=%s", p)
	}
	return b.String()
}

func (l defaultLinker) linkWholeArchives(libs []string) string {
	if len(libs) == 0 {
		return ""
	}
	return fmt.Sprintf("-Wl,--whole-archive %s -Wl,--no-whole-archive", utils.Join(libs))
}

func newDefaultLinker(tool string, flags, libs []string) (linker defaultLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}

type toolchain interface {
	getArchiver() (tool string, flags []string)
	getAssembler() (tool string, flags []string)
	getCCompiler() (tool string, flags []string)
	getCXXCompiler() (tool string, flags []string)
	getLinker() linker
	getStripBinary() (tool string)
}

func lookPathSecond(toolUnqualified string, firstHit string) (string, error) {
	firstDir := filepath.Clean(filepath.Dir(firstHit))
	// In the Soong plugin, this is the only environment variable reference. The Soong plugin
	// does not hash the environment, so if it were any other variable, there would be a
	// missing dependency. Fortunately, Soong itself keeps track of PATH, and will
	// automatically regenerate the Ninja file when it changes, so accessing it here is safe.
	path := os.Getenv("PATH")
	foundFirstHit := false
	for _, dir := range filepath.SplitList(path) {
		if foundFirstHit {
			if fname := filepath.Join(dir, toolUnqualified); utils.IsExecutable(fname) {
				return fname, nil
			}
		} else if filepath.Clean(dir) == firstDir {
			foundFirstHit = true
		}
	}
	return "", &exec.Error{Name: toolUnqualified, Err: exec.ErrNotFound}
}

func getToolPath(toolUnqualified string) (toolPath string) {
	if filepath.IsAbs(toolUnqualified) {
		toolPath = toolUnqualified
		toolUnqualified = filepath.Base(toolUnqualified)
	} else {
		path, err := exec.LookPath(toolUnqualified)
		if err != nil {
			panic(fmt.Errorf("Error: Couldn't get tool from path %s: %v", toolUnqualified, err))
		}
		toolPath = path
	}

	// If the target is a ccache symlink, try the lookup again, but
	// ignoring the directory in PATH that the symlink was found in.
	if fi, err := os.Lstat(toolPath); err == nil && (fi.Mode()&os.ModeSymlink != 0) {
		linkTarget, err := os.Readlink(toolPath)
		if err == nil && filepath.Base(linkTarget) == "ccache" {
			toolPath, err = lookPathSecond(toolUnqualified, toolPath)
			if err != nil {
				panic(fmt.Errorf("%s is a ccache symlink, and could not find the actual binary",
					toolPath))
			}
		}
	}
	return
}

// Run the compiler with the -print-file-name option, and return the result.
// Check that the file exists. Return an error if the file can't be located.
func getFileName(tc toolchain, basename string) (fname string, e error) {
	ccBinary, flags := tc.getCCompiler()

	cmd := exec.Command(ccBinary, utils.NewStringSlice(flags, []string{"-print-file-name=" + basename})...)
	bytes, err := cmd.Output()
	if err != nil {
		e = fmt.Errorf("Couldn't get path for %s: %v", basename, err)
		return
	}

	fname = strings.TrimSpace(string(bytes))

	if _, err := os.Stat(fname); os.IsNotExist(err) {
		e = fmt.Errorf("Path returned for %s (%s) does not exist", basename, fname)
		return
	}

	return
}

// Run the compiler with the -print-file-name option, and return the directory
// name of the result, or an empty list if a non-existent directory was
// returned or an error occurred.
func getFileNameDir(tc toolchain, basename string) (dirs []string) {

	fname, err := getFileName(tc, basename)

	if err == nil {
		dirs = append(dirs, filepath.Dir(fname))
	} else {
		fmt.Printf("Error: %s\n", err.Error())
	}

	return
}

type toolchainGnu interface {
	toolchain
	getBinDirs() []string
	getStdCxxHeaderDirs() []string
	getInstallDir() string
}

type toolchainGnuCommon struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	gccBinary     string
	gxxBinary     string
	linker        linker
	prefix        string
	cflags        []string // Flags for both C and C++
	ldflags       []string // Linker flags, including anything required for C++
	binDir        string
}

type toolchainGnuNative struct {
	toolchainGnuCommon
}

type toolchainGnuCross struct {
	toolchainGnuCommon
}

func (tc toolchainGnuCommon) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainGnuCommon) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainGnuCommon) getCCompiler() (string, []string) {
	return tc.gccBinary, tc.cflags
}

func (tc toolchainGnuCommon) getCXXCompiler() (tool string, flags []string) {
	return tc.gxxBinary, tc.cflags
}

func (tc toolchainGnuCommon) getLinker() linker {
	return tc.linker
}

func (tc toolchainGnuCommon) getStripBinary() string {
	return tc.objcopyBinary
}

func (tc toolchainGnuCommon) getBinDirs() []string {
	return []string{tc.binDir}
}

// The libstdc++ headers shipped with GCC toolchains are stored, relative to
// the `prefix-gcc` binary's location, in `../$ARCH/include/c++/$VERSION` and
// `../$ARCH/include/c++/$VERSION/$ARCH`. This function returns $ARCH. This is
// generally the same as the compiler prefix, but because the prefix can
// contain the path to the compiler as well, we instead obtain it by trying the
// `-print-multiarch` and `-dumpmachine` options.
func (tc toolchainGnuCommon) getTargetTripleHeaderSubdir() string {
	ccBinary, flags := tc.getCCompiler()
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
	ccBinary, _ := tc.getCCompiler()
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

func newToolchainGnuCommon(config *bobConfig, tgt tgtType) (tc toolchainGnuCommon) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_gnu_prefix")
	tc.arBinary = tc.prefix + props.GetString("ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")

	tc.gccBinary = tc.prefix + props.GetString("gnu_cc_binary")
	tc.gxxBinary = tc.prefix + props.GetString("gnu_cxx_binary")
	tc.binDir = filepath.Dir(getToolPath(tc.gccBinary))

	sysroot := props.GetString(string(tgt) + "_sysroot")
	if sysroot != "" {
		tc.cflags = append(tc.cflags, "--sysroot="+sysroot)
		tc.ldflags = append(tc.ldflags, "--sysroot="+sysroot)
	}

	flags := strings.Split(config.Properties.GetString(string(tgt)+"_gnu_flags"), " ")
	tc.cflags = append(tc.cflags, flags...)
	tc.ldflags = append(tc.ldflags, flags...)

	tc.linker = newDefaultLinker(tc.gxxBinary, tc.ldflags, []string{})

	return
}

func newToolchainGnuNative(config *bobConfig) (tc toolchainGnuNative) {
	tc.toolchainGnuCommon = newToolchainGnuCommon(config, tgtTypeHost)
	return
}

func newToolchainGnuCross(config *bobConfig) (tc toolchainGnuCross) {
	tc.toolchainGnuCommon = newToolchainGnuCommon(config, tgtTypeTarget)
	return
}

type toolchainClangCommon struct {
	// Options read from the config:
	arBinary       string
	asBinary       string
	objcopyBinary  string
	clangBinary    string
	clangxxBinary  string
	linker         linker
	prefix         string
	useGnuBinutils bool

	// Use the GNU toolchain's 'ar' and 'as', as well as its libstdc++
	// headers if required
	gnu toolchainGnu

	// Calculated during toolchain initialization:
	cflags   []string // Flags for both C and C++
	cxxflags []string // Flags just for C++
	ldflags  []string // Linker flags, including anything required for C++
	ldlibs   []string // Linker libraries

	target string
}

type toolchainClangNative struct {
	toolchainClangCommon
}

type toolchainClangCross struct {
	toolchainClangCommon
}

func (tc toolchainClangCommon) getArchiver() (string, []string) {
	if tc.useGnuBinutils {
		return tc.gnu.getArchiver()
	}
	return tc.arBinary, []string{}
}

func (tc toolchainClangCommon) getAssembler() (string, []string) {
	if tc.useGnuBinutils {
		return tc.gnu.getAssembler()
	}
	return tc.asBinary, []string{}
}

func (tc toolchainClangCommon) getCCompiler() (string, []string) {
	return tc.clangBinary, tc.cflags
}

func (tc toolchainClangCommon) getCXXCompiler() (string, []string) {
	return tc.clangxxBinary, tc.cxxflags
}

func (tc toolchainClangCommon) getLinker() linker {
	return newDefaultLinker(tc.clangxxBinary, tc.ldflags, tc.ldlibs)
}

func (tc toolchainClangCommon) getStripBinary() string {
	return tc.objcopyBinary
}

func newToolchainClangCommon(config *bobConfig, tgt tgtType) (tc toolchainClangCommon) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_clang_prefix")

	// This assumes arBinary and asBinary are either in the path, or the same directory as clang.
	// This is not necessarily the case. This will need to be updated when we support clang on linux without a GNU toolchain.
	tc.arBinary = tc.prefix + props.GetString("ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")

	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")

	tc.clangBinary = tc.prefix + props.GetString("clang_cc_binary")
	tc.clangxxBinary = tc.prefix + props.GetString("clang_cxx_binary")

	tc.target = props.GetString(string(tgt) + "_clang_triple")

	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
		tc.ldflags = append(tc.ldflags, "-target", tc.target)
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
		if tgt == tgtTypeHost {
			tc.gnu = newToolchainGnuNative(config)
		} else {
			tc.gnu = newToolchainGnuCross(config)
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
		// Tell Clang where the GNU toolchain is installed, so it can use its
		// headers and libraries, for example, if we are using libstdc++.
		gnuInstallArg := "--gcc-toolchain=" + tc.gnu.getInstallDir()
		tc.cflags = append(tc.cflags, gnuInstallArg)
		tc.ldflags = append(tc.ldflags, gnuInstallArg)
	}
	if useGnuCrt {
		binDirs = append(binDirs, getFileNameDir(tc.gnu, "crt1.o")...)
	}
	if tc.useGnuBinutils {
		// Add the GNU toolchain's binary directories to Clang's binary search
		// path, so that Clang can find the correct linker. If the GNU toolchain
		// is a "system" toolchain (e.g. in /usr/bin), its binaries will already
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
	// every call to getCXXCompiler().
	tc.cxxflags = append(tc.cxxflags, tc.cflags...)

	tc.linker = newDefaultLinker(tc.clangxxBinary, tc.cflags, []string{})

	return
}

func newToolchainClangNative(config *bobConfig) (tc toolchainClangNative) {
	tc.toolchainClangCommon = newToolchainClangCommon(config, tgtTypeHost)
	return
}

func newToolchainClangCross(config *bobConfig) (tc toolchainClangCross) {
	tc.toolchainClangCommon = newToolchainClangCommon(config, tgtTypeTarget)
	return
}

type toolchainArmClang struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	ccBinary      string
	cxxBinary     string
	linker        linker
	prefix        string
	cflags        []string // Flags for both C and C++
}

type toolchainArmClangNative struct {
	toolchainArmClang
}

type toolchainArmClangCross struct {
	toolchainArmClang
}

func (tc toolchainArmClang) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainArmClang) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainArmClang) getCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainArmClang) getCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

func (tc toolchainArmClang) getLinker() linker {
	return tc.linker
}

func (tc toolchainArmClang) getStripBinary() string {
	return tc.objcopyBinary
}

func newToolchainArmClangCommon(config *bobConfig, tgt tgtType) (tc toolchainArmClang) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_gnu_prefix")
	tc.arBinary = tc.prefix + props.GetString("armclang_ar_binary")
	tc.asBinary = tc.prefix + props.GetString("armclang_as_binary")
	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")
	tc.ccBinary = tc.prefix + props.GetString("armclang_cc_binary")
	tc.cxxBinary = tc.prefix + props.GetString("armclang_cxx_binary")
	tc.linker = newDefaultLinker(tc.cxxBinary, []string{}, []string{})

	tc.cflags = strings.Split(config.Properties.GetString(string(tgt)+"_armclang_flags"), " ")

	return
}

func newToolchainArmClangNative(config *bobConfig) (tc toolchainArmClangNative) {
	tc.toolchainArmClang = newToolchainArmClangCommon(config, tgtTypeHost)
	return
}

func newToolchainArmClangCross(config *bobConfig) (tc toolchainArmClangCross) {
	tc.toolchainArmClang = newToolchainArmClangCommon(config, tgtTypeTarget)
	return
}

type toolchainXcode struct {
	arBinary      string
	asBinary      string
	objcopyBinary string
	ccBinary      string
	cxxBinary     string
	linker        linker
	prefix        string
	target        string

	cflags  []string
	ldflags []string
}

type toolchainXcodeNative struct {
	toolchainXcode
}

type toolchainXcodeCross struct {
	toolchainXcode
}

func (tc toolchainXcode) getArchiver() (string, []string) {
	return tc.arBinary, []string{}
}

func (tc toolchainXcode) getAssembler() (string, []string) {
	return tc.asBinary, []string{}
}

func (tc toolchainXcode) getCCompiler() (string, []string) {
	return tc.ccBinary, tc.cflags
}

func (tc toolchainXcode) getCXXCompiler() (string, []string) {
	return tc.cxxBinary, tc.cflags
}

type xcodeLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l xcodeLinker) getTool() string {
	return l.tool
}

func (l xcodeLinker) getFlags() []string {
	return l.flags
}

func (l xcodeLinker) getLibs() []string {
	return l.libs
}

func (l xcodeLinker) keepUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) dropUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) setRpathLink(path string) string {
	return ""
}

func (l xcodeLinker) setRpath(path []string) string {
	return ""
}

func (l xcodeLinker) linkWholeArchives(libs []string) string {
	return utils.Join(libs)
}

func (l xcodeLinker) keepSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) dropSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) getForwardingLibFlags() string {
	return ""
}

func newXcodeLinker(tool string, flags, libs []string) (linker xcodeLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}

func (tc toolchainXcode) getLinker() linker {
	return tc.linker
}

func (tc toolchainXcode) getStripBinary() string {
	return tc.objcopyBinary
}

func newToolchainXcodeCommon(config *bobConfig, tgt tgtType) (tc toolchainXcode) {
	props := config.Properties
	tc.prefix = props.GetString(string(tgt) + "_xcode_prefix")
	tc.arBinary = tc.prefix + props.GetString("ar_binary")
	tc.asBinary = tc.prefix + props.GetString("as_binary")
	tc.objcopyBinary = props.GetString(string(tgt) + "_objcopy_binary")

	tc.ccBinary = tc.prefix + props.GetString("clang_cc_binary")
	tc.cxxBinary = tc.prefix + props.GetString("clang_cxx_binary")

	tc.target = props.GetString(string(tgt) + "_xcode_triple")

	if tc.target != "" {
		tc.cflags = append(tc.cflags, "-target", tc.target)
		tc.ldflags = append(tc.ldflags, "-target", tc.target)
	}

	tc.linker = newXcodeLinker(tc.cxxBinary, tc.ldflags, []string{})

	return
}

func newToolchainXcodeNative(config *bobConfig) (tc toolchainXcodeNative) {
	tc.toolchainXcode = newToolchainXcodeCommon(config, tgtTypeHost)
	return
}

func newToolchainXcodeCross(config *bobConfig) (tc toolchainXcodeCross) {
	tc.toolchainXcode = newToolchainXcodeCommon(config, tgtTypeTarget)
	return
}

type toolchainSet struct {
	host   toolchain
	target toolchain
}

func (tcs *toolchainSet) getToolchain(tgt tgtType) toolchain {
	if tgt == tgtTypeHost {
		return tcs.host
	}
	return tcs.target
}

func (tcs *toolchainSet) parseConfig(config *bobConfig) {
	props := config.Properties

	if props.GetBool("target_toolchain_clang") {
		tcs.target = newToolchainClangCross(config)
	} else if props.GetBool("target_toolchain_gnu") {
		tcs.target = newToolchainGnuCross(config)
	} else if props.GetBool("target_toolchain_armclang") {
		tcs.target = newToolchainArmClangCross(config)
	} else if props.GetBool("target_toolchain_xcode") {
		tcs.target = newToolchainXcodeCross(config)
	} else {
		panic(errors.New("no usable target compiler toolchain configured"))
	}

	if props.GetBool("host_toolchain_clang") {
		tcs.host = newToolchainClangNative(config)
	} else if props.GetBool("host_toolchain_gnu") {
		tcs.host = newToolchainGnuNative(config)
	} else if props.GetBool("host_toolchain_armclang") {
		tcs.host = newToolchainArmClangNative(config)
	} else if props.GetBool("host_toolchain_xcode") {
		tcs.host = newToolchainXcodeNative(config)
	} else {
		panic(errors.New("no usable host compiler toolchain configured"))
	}
}
