//go:build soong
// +build soong

package soong_compat

import (
	"android/soong/android"
)

// This definition is compatible with Soong SHAs after `aa2555387 Add ctx to
// AndroidMkExtraEntriesFunc`
func ConvertAndroidMkExtraEntriesFunc(f AndroidMkExtraEntriesFunc) []android.AndroidMkExtraEntriesFunc {
	return []android.AndroidMkExtraEntriesFunc{
		func(ctx android.AndroidMkExtraEntriesContext, entries *android.AndroidMkEntries) {
			f(entries)
		},
	}
}

func SoongSupportsMkInstallTargets() bool {
	return false
}
