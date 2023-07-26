//go:build soong
// +build soong

package soong_compat

import (
	"android/soong/android"
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
