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
// `c93979f60d79f51d8432d50f5f5eac2ff347fe91
// Add ModuleProxy that should be used when visiting deps.`
func GetHostBinPath(ctx android.ModuleContext, m android.ModuleOrProxy, host_bin string) android.OptionalPath {
	htp, ok := android.OtherModuleProvider(ctx, m, android.HostToolProviderInfoProvider)

	if !ok {
		panic(fmt.Errorf("No HostToolProviderInfoProvider for %s module!", host_bin))
	}

	return htp.HostToolPath
}
