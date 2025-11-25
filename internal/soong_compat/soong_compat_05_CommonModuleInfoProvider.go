//go:build soong
// +build soong

package soong_compat

import (
	"android/soong/android"
	"fmt"
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
	return true
}

// This definition is compatible with Soong SHAs _after_
// `f22120fb1da7f75a571966124bb0da6a57fd4f07 Change CommonModuleInfoProvider to a pointer.`
func GetHostBinPath(ctx android.ModuleContext, m android.ModuleOrProxy, host_bin string) android.OptionalPath {
	if p, ok := android.OtherModuleProvider(ctx, m, android.CommonModuleInfoProvider); ok && p.HostToolInfo != nil {
		return p.HostToolInfo.HostToolPath
	} else {
		panic(fmt.Errorf("No CommonModuleInfoProvider for %s module!", host_bin))
	}

	return android.OptionalPath{}
}
