package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/internal/bpwriter"
	"github.com/ARM-software/bob-build/internal/utils"
)

func writeDataResourceModule(m bpwriter.Module, src, installRel string) {
	// add prebuilt_etc properties
	m.AddString("src", src)
	m.AddString("sub_dir", installRel)
	m.AddBool("filename_from_src", true)
	m.AddBool("installable", true)
}

func writeCodeResourceModule(m bpwriter.Module, src, installRel string) {
	m.AddStringList("srcs", []string{src})
	m.AddString("stem", filepath.Base(src))
	m.AddString("relative_install_path", installRel)
}

func (m *ModuleResource) getAndroidbpResourceName(src string) string {
	return m.shortName() + "__" + strings.Replace(src, "/", "_", -1)
}

func (g *androidBpGenerator) resourceActions(r *ModuleResource, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(r) {
		return
	}

	installBase, installRel, _ := getSoongInstallPath(r.getInstallableProps())

	var modType string
	// Soong has two types of backend modules; "data" ones, for places like
	// /etc, and "code" ones, for locations like /bin. Write different sets
	// of properties depending on which one is required.
	var write func(bpwriter.Module, string, string)

	if installBase == "data" {
		modType = "prebuilt_data_bob"
		write = writeDataResourceModule
	} else if installBase == "etc" {
		modType = "prebuilt_etc"
		write = writeDataResourceModule
	} else if installBase == "firmware" {
		modType = "prebuilt_firmware"
		write = writeDataResourceModule
	} else if installBase == "bin" {
		modType = "cc_prebuilt_binary"
		write = writeCodeResourceModule
	} else if installBase == "tests" {
		// Eventually we want to install in testcases,
		// But we can't put binaries there yet.
		// So place resources in /data/nativetest to align with cc_test.
		//modType = "prebuilt_testcase_bob"
		modType = "prebuilt_data_bob"
		if r.Properties.isProprietary() {
			// Vendor modules need an additional path element to match cc_test
			installRel = filepath.Join("nativetest", "vendor", installRel)
		} else {
			installRel = filepath.Join("nativetest", installRel)
		}
		write = writeDataResourceModule
	} else {
		panic(fmt.Errorf("Could not detect partition for install path '%s'", installBase))
	}

	r.Properties.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			// keep module name unique, remove slashes
			m, err := AndroidBpFile().NewModule(modType, r.getAndroidbpResourceName(fp.UnScopedPath()))
			if err != nil {
				utils.Die(err.Error())
			}

			addProvenanceProps(m, r.Properties.AndroidProps)

			write(m, fp.UnScopedPath(), installRel)
			return true
		})

}
