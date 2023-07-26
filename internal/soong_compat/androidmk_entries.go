//go:build soong
// +build soong

package soong_compat

import (
	"android/soong/android"
)

type AndroidMkExtraEntriesFunc = func(*android.AndroidMkEntries)
