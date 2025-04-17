//go:build soong
// +build soong

package soong_compat

import (
	"android/soong/android"
	"fmt"
	"github.com/google/blueprint"
)

// This definition is compatible with Soong SHAs _before_ `aa2555387 Add ctx to
// AndroidMkExtraEntriesFunc` It requires Soong SHA `0b0e1b980 AndroidMkEntries()
// returns multiple AndroidMkEntries structs` or later.
func ConvertAndroidMkExtraEntriesFunc(f AndroidMkExtraEntriesFunc) []android.AndroidMkExtraEntriesFunc {
	return []android.AndroidMkExtraEntriesFunc{
		func(entries *android.AndroidMkEntries) {
			f(entries)
		},
	}
}

func SoongSupportsMkInstallTargets() bool {
	return false
}

// This definition is compatible with Soong SHAs _before_
// `dd9ccb4234dfc88a004e36b2c0500769a5f50ad3
// Add ModuleProxy that should be used when visiting deps.`
func GetHostBinPath(ctx android.ModuleContext, m blueprint.Module, host_bin string) android.OptionalPath {
	htp, ok := m.(android.HostToolProvider)

	if !ok {
		panic(fmt.Errorf("%s is not a host tool", host_bin))
	}

	return htp.HostToolPath()
}
