package toolchain

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ARM-software/bob-build/core/config"
	"github.com/ARM-software/bob-build/internal/utils"
)

type TgtType string

const (
	TgtTypeHost    TgtType = "host"
	TgtTypeTarget  TgtType = "target"
	TgtTypeUnknown TgtType = ""
)

type Toolchain interface {
	GetArchiver() (tool string, flags []string)
	GetAssembler() (tool string, flags []string)
	GetCCompiler() (tool string, flags []string)
	GetCXXCompiler() (tool string, flags []string)
	GetLinker() Linker
	GetStripFlags() []string
	GetLibraryTocFlags() []string
	CheckFlagIsSupported(language, flag string) bool
	Is64BitOnly() bool
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

func getToolPath(toolUnqualified string) string {
	var toolPath string

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

	// Follow symlinks to get to the actual tool location, in case e.g. it
	// is going via something like update-alternatives.
	realToolPath, err := filepath.EvalSymlinks(toolPath)
	if err != nil {
		panic(fmt.Errorf("Could not follow Toolchain symlink %s: %v", toolPath, err))
	}

	return realToolPath
}

// Run the compiler with the -print-file-name option, and return the result.
// Check that the file exists. Return an error if the file can't be located.
func getFileName(tc Toolchain, basename string) (fname string, e error) {
	ccBinary, flags := tc.GetCCompiler()

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
func getFileNameDir(tc Toolchain, basename string) (dirs []string) {

	fname, err := getFileName(tc, basename)

	if err == nil {
		dirs = append(dirs, filepath.Dir(fname))
	} else {
		fmt.Printf("Error: %s\n", err.Error())
	}

	return
}

// TODO: move to utils
// Type for caching the supported flags of a compiler
// Cache maps flags+compiler+language to an boolean:
//
//	false - not supported
//	true  - supported
type flagSupportedCache struct {
	m    map[string]bool
	lock sync.RWMutex
}

func newFlagCache() (cache *flagSupportedCache) {
	cache = &flagSupportedCache{}
	cache.m = make(map[string]bool)
	return
}

// Check that a Toolchain's compiler for 'language' supports the given 'flag'
func (cache *flagSupportedCache) checkFlag(tc Toolchain, language, flag string) bool {
	compiler := ""
	flags := []string{}
	switch language {
	case "c++":
		compiler, flags = tc.GetCXXCompiler()
	case "c":
		compiler, flags = tc.GetCCompiler()
	default:
		// No other language currently supported
		return false
	}

	// The search key is "<flag>/<compiler>/<language>"
	key := strings.Join([]string{flag, compiler, language}, "/")

	cache.lock.RLock()
	supported, ok := cache.m[key]
	cache.lock.RUnlock()
	if ok {
		return supported
	}

	// We have not seen the flag before, check it by running the compiler with the flag
	// Add a '-Werror' to make sure that the compiler exits with an error code if the
	// flag is unknown. If the flag starts with '-Wno-' remove the 'no-' part so that
	// we can test the actual flag. This is to work around the fact that gcc is silent
	// about '-Wno-<flag_name>' flags it doesn't recognise until you actually compile a file
	saneFlag := strings.Replace(flag, "-Wno-", "-W", 1)
	testFlags := utils.NewStringSlice(flags, []string{"-x", language, "-c", os.DevNull, "-o", os.DevNull, "-Werror", saneFlag})
	testFlags = utils.Remove(testFlags, "")
	cmd := exec.Command(compiler, testFlags...)
	_, err := cmd.CombinedOutput()
	if err == nil {
		cache.lock.Lock()
		cache.m[key] = true
		cache.lock.Unlock()
		return true
	}

	// Compiler did not recognise the flag
	cache.lock.Lock()
	cache.m[key] = false
	cache.lock.Unlock()

	return false
}

type ToolchainSet struct {
	host   Toolchain
	target Toolchain
}

func (tcs *ToolchainSet) GetToolchain(tgt TgtType) Toolchain {
	if tgt == TgtTypeHost {
		return tcs.host
	}
	return tcs.target
}

func (tcs *ToolchainSet) Configure(props *config.Properties) {

	if props.GetBool("target_toolchain_clang") {
		tcs.target = newToolchainClangCross(props)
	} else if props.GetBool("target_toolchain_gnu") {
		tcs.target = newToolchainGnuCross(props)
	} else if props.GetBool("target_toolchain_armclang") {
		tcs.target = newToolchainArmClangCross(props)
	} else if props.GetBool("target_toolchain_xcode") {
		tcs.target = newToolchainXcodeCross(props)
	} else {
		panic(errors.New("no usable target compiler Toolchain configured"))
	}

	if props.GetBool("host_toolchain_clang") {
		tcs.host = newToolchainClangNative(props)
	} else if props.GetBool("host_toolchain_gnu") {
		tcs.host = newToolchainGnuNative(props)
	} else if props.GetBool("host_toolchain_armclang") {
		tcs.host = newToolchainArmClangNative(props)
	} else if props.GetBool("host_toolchain_xcode") {
		tcs.host = newToolchainXcodeNative(props)
	} else {
		panic(errors.New("no usable host compiler Toolchain configured"))
	}
}
