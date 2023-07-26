//go:build soong
// +build soong

package prebuilt

import (
	"android/soong/android"
	"android/soong/etc"

	"github.com/ARM-software/bob-build/internal/soong_compat"
)

func init() {
	android.RegisterModuleType("prebuilt_data_bob", PrebuiltDataFactory)
	android.RegisterModuleType("prebuilt_testcase_bob", PrebuiltTestcaseFactory)
}

// Prebuilts going to data partition
type PrebuiltData struct {
	// pretend to be soong prebuilt module
	etc.PrebuiltEtc
}

// implemented interfaces check
var _ android.Module = (*PrebuiltData)(nil)
var _ android.AndroidMkEntriesProvider = (*PrebuiltData)(nil)

func PrebuiltDataFactory() android.Module {
	m := &PrebuiltData{}
	// register PrebuiltEtc properties,
	// install path will be relative to data partition root
	etc.InitPrebuiltEtcModule(&m.PrebuiltEtc, "")

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibFirst)

	return m
}

func (m *PrebuiltData) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "DATA",
		OutputFile: android.OptionalPathForPath(m.OutputFile()),
		Include:    "$(BUILD_PREBUILT)",
		ExtraEntries: soong_compat.ConvertAndroidMkExtraEntriesFunc(
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", m.InstallDirPath().ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", m.OutputFile().Base())
			},
		),
	}}
}

// required to generate ninja rule for copying files onto data partition
func (m *PrebuiltData) InstallInData() bool {
	return true
}

// Prebuilts going to data partition
type PrebuiltTestcase struct {
	// pretend to be soong prebuilt module
	etc.PrebuiltEtc
}

// implemented interfaces check
var _ android.Module = (*PrebuiltTestcase)(nil)
var _ android.AndroidMkEntriesProvider = (*PrebuiltTestcase)(nil)

func PrebuiltTestcaseFactory() android.Module {
	m := &PrebuiltTestcase{}
	// register PrebuiltEtc properties,
	// install path will be relative to the `testcases` directory
	etc.InitPrebuiltEtcModule(&m.PrebuiltEtc, "")

	// init module (including name and common properties) with target-specific variants info
	android.InitAndroidArchModule(m, android.DeviceSupported, android.MultilibFirst)

	return m
}

func (m *PrebuiltTestcase) AndroidMkEntries() []android.AndroidMkEntries {
	return []android.AndroidMkEntries{{
		Class:      "DATA",
		OutputFile: android.OptionalPathForPath(m.OutputFile()),
		Include:    "$(BUILD_PREBUILT)",
		ExtraEntries: soong_compat.ConvertAndroidMkExtraEntriesFunc(
			func(entries *android.AndroidMkEntries) {
				entries.SetString("LOCAL_MODULE_PATH", m.InstallDirPath().ToMakePath().String())
				entries.SetString("LOCAL_INSTALLED_MODULE_STEM", m.OutputFile().Base())
			},
		),
	}}
}

// required to generate ninja rule for copying files into the `testcases` directory
func (m *PrebuiltTestcase) InstallInTestcases() bool {
	return true
}
