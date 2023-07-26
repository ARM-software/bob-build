package ccflags

// Encapsulate knowledge about common compiler and linker flags

import (
	"fmt"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
)

// This flag is a machine specific option
func machineSpecificFlag(s string) bool {
	return strings.HasPrefix(s, "-m")
}

// This flag selects the compiler standard
func compilerStandard(s string) bool {
	return strings.HasPrefix(s, "-std=")
}

func thumbFlag(s string) bool {
	return s == "-mthumb"
}

func armFlag(s string) bool {
	return s == "-marm" || s == "-mno-thumb"
}

// Identify whether a compilation flag should be used on android
//
// The Android build system should set machine specific flags (so it
// can do multi-arch builds) and compiler standard, so filter these
// out from module properties.
func AndroidCompileFlags(s string) bool {
	return !(machineSpecificFlag(s) || compilerStandard(s))
}

// Identify whether a link flag should be used on android
//
// The Android build system should set machine specific flags (so it
// can do multi-arch builds), so filter these out from module
// properties.
func AndroidLinkFlags(s string) bool {
	return !machineSpecificFlag(s)
}

func GetCompilerStandard(flags ...[]string) (std string) {
	// Look for the flag setting compiler standard
	stdList := utils.Filter(compilerStandard, flags...)
	if len(stdList) > 0 {
		// Use last definition only
		std = strings.TrimPrefix(stdList[len(stdList)-1], "-std=")
	}
	return
}

func GetArmMode(flags ...[]string) (armMode string, err error) {
	// Look for the flag setting thumb or not thumb
	thumb := utils.Filter(thumbFlag, flags...)
	arm := utils.Filter(armFlag, flags...)
	if len(thumb) > 0 && len(arm) > 0 {
		err = fmt.Errorf("Both thumb and no thumb (arm) options are specified")
	} else if len(thumb) > 0 {
		armMode = "thumb"
	} else if len(arm) > 0 {
		armMode = "arm"
	}
	return
}
