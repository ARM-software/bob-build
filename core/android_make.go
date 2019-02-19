/*
 * Copyright 2018 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/blueprint"

	"github.com/ARM-software/bob-build/utils"
)

const (
	binTypeStatic     = iota
	binTypeShared     = iota
	binTypeExecutable = iota
)

type androidMkGenerator struct {
	toolchainSet
}

var androidLock sync.Mutex

func writeIfChanged(filename string, text string) {
	mustWrite := true

	// If any errors occur trying to determine the state of the existing file,
	// just write the new file
	fileinfo, err := os.Stat(filename)
	if err == nil {
		if fileinfo.Size() == int64(len(text)) {
			current, err := ioutil.ReadFile(filename)
			if err == nil {
				if string(current) == text {
					// No need to write
					mustWrite = false
				}
			}
		}
	}

	if mustWrite {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			panic(err)
		}

		file.WriteString(text)
		file.Close()
	}
}

func androidMkWriteString(ctx blueprint.ModuleContext, name string, text string) {
	filename := filepath.Join(builddir, name+".inc")
	writeIfChanged(filename, text)
}

func newlineSeparatedList(list []string) string {
	return " \\\n    " + strings.Join(list, " \\\n    ") + "\n"
}

// This flag is a machine specific option
func machineSpecificFlag(s string) bool {
	return strings.HasPrefix(s, "-m")
}

// This flag selects the compiler standard
func compilerStandard(s string) bool {
	return strings.HasPrefix(s, "-std=")
}

// Identify whether a compilation flag should be used on android
//
// The Android build system should set machine specific flags (so it
// can do multi-arch builds) and compiler standard, so filter these
// out from module properties.
func moduleCompileFlags(s string) bool {
	return !(machineSpecificFlag(s) || compilerStandard(s))
}

// Identify whether a link flag should be used on android
//
// The Android build system should set machine specific flags (so it
// can do multi-arch builds), so filter these out from module
// properties.
func moduleLinkFlags(s string) bool {
	return !machineSpecificFlag(s)
}

// If we see any of there libraries in LDLIBS as -lxxx, we'll replace them with the android library object
var androidSharedLibs = [...]string{
	"libbinder",
	"libc++",
	"libcutils",
	"libgui",
	"libhardware",
	"libion",
	"liblog",
	"libnativewindow",
	"libsync",
	"libui",
	"libutils"}

var androidStaticLibs = [...]string{
	"libarect"}

var androidHeaderLibs = [...]string{
	"libcutils_headers",
	"libgui_headers",
	"libhardware_headers",
	"liblog_headers",
	"libnativebase_headers",
	"libsystem_headers",
	"libui_headers"}

func noAndroidLdlibs(s string) bool {
	// An android module should have a way to add android libraries to its LOCAL_SHARED_LIBRARIES,
	// LOCAL_HEADER_LIBRARIES and LOCAL_STATIC_LIBRARIES. The way we do this is via ldlibs, if a
	// module has a definition in ldlibs which doesn't pass this predicate, it is this kind of
	// library. In this case, module is removed from LOCAL_LDLIBS and instead added to either,
	// LOCAL_SHARED_LIBS, LOCAL_HEADER_LIBS or LOCAL_STATIC_LIBS.
	if strings.HasPrefix(s, "android.") {
		return false
	}
	for _, lib := range androidSharedLibs {
		if s[2:] == lib[3:] {
			// Don't include android shared lib
			return false
		}
	}
	for _, lib := range androidHeaderLibs {
		if s[2:] == lib[3:] {
			// Don't include android header lib
			return false
		}
	}
	for _, lib := range androidStaticLibs {
		if s[2:] == lib[3:] {
			// Don't include android static lib
			return false
		}
	}
	return true
}

var (
	classes = []string{
		"STATIC_LIBRARIES",
		"SHARED_LIBRARIES",
		"EXECUTABLES",
	}

	rulePrefix = map[string]string{
		tgtTypeTarget: "BUILD_",
		tgtTypeHost:   "BUILD_HOST_",
	}

	ruleSuffix = []string{
		"STATIC_LIBRARY",
		"SHARED_LIBRARY",
		"EXECUTABLE",
	}
)

func specifyCompilerStandard(varname string, flags []string) string {
	// Look for the flag setting compiler standard
	line := ""
	stdList := utils.Filter(flags, compilerStandard)
	if len(stdList) > 0 {
		// Use last definition only
		std := strings.TrimPrefix(stdList[len(stdList)-1], "-std=")
		line += varname + ":=" + std + "\n"
	}
	return line
}

func (m *library) GenerateBuildAction(binType int, ctx blueprint.ModuleContext) {
	if m.Properties.Build_wrapper != nil {
		panic(errors.New("build_wrapper not supported on Android"))
	}

	text := "##########################\ninclude $(CLEAR_VARS)\n\n"
	text += "LOCAL_MODULE:=" + m.altName() + "\n"
	text += "LOCAL_MODULE_CLASS:=" + classes[binType] + "\n\n"

	// The order we want is  local_include_dirs, export_local_include_dirs,
	//                       include_dirs, export_include_dirs
	// This is because include and export include should be system headers
	includes := utils.PrefixDirs(m.Properties.Local_include_dirs, "$(LOCAL_PATH)")
	includes = append(includes, utils.PrefixDirs(m.Properties.Export_local_include_dirs, "$(LOCAL_PATH)")...)
	includes = append(includes, m.Properties.Include_dirs...)
	includes = append(includes, m.Properties.Export_include_dirs...)

	exportIncludeDirs := utils.NewStringSlice(m.Properties.Export_include_dirs, utils.PrefixDirs(m.Properties.Export_local_include_dirs, "$(LOCAL_PATH)"))

	// Handle generated headers
	if len(m.Properties.Generated_headers) > 0 {
		headerDirs, headerOutputs := m.GetGeneratedHeaders(ctx)
		includes = append(includes, headerDirs...)
		headers := strings.Join(headerOutputs, " ")

		text += "LOCAL_ADDITIONAL_DEPENDENCIES := " + headers + "\n\n"
	}

	// Handle generated sources
	for _, module := range m.Properties.Generated_sources {
		// LOCAL_GENERATED_SOURCES is used to name target generated as
		// part of this module which we also link into a library. The
		// generated sources are automatically added to the
		// library. Unfortunately, we've generated the sources in a
		// separate module...
		//
		// To compile a file generated in another module we could try
		// and explicitly list the file in that modules directory in
		// LOCAL_SRCS. However Android make won't work with LOCAL_SRCS
		// outside the source tree, so we can't do that.
		//
		// Therefore we use LOCAL_GENERATED_SOURCES and copy the files
		// generated in the other module into this module.
		//
		// An alternative would be to avoid the separate source
		// generation module and do it as part of this module.

		sources := "$(" + module + "_OUTPUTS)"
		sourcesDir := "$(" + module + "_GEN_DIR)"

		localSourceExpr := "$(subst " + sourcesDir + ", $(local-generated-sources-dir), " + sources + ")"
		localSources := "$(" + m.altName() + "_" + module + "_SRCS)"

		text += m.altName() + "_" + module + "_SRCS:=" + localSourceExpr + "\n"
		text += "LOCAL_GENERATED_SOURCES+=" + localSources + "\n"

		// Copy rule. Use a static pattern to avoid running the command for each file
		text += localSources + ": $(local-generated-sources-dir)" + "/%: " + sourcesDir + "/%\n"
		text += "\tcp $< $@\n\n"
	}

	if getConfig(ctx).Properties.GetBool("target_toolchain_clang") {
		text += "LOCAL_CLANG := true\n"
	} else {
		text += "LOCAL_CLANG := false\n"
	}
	srcs := utils.NewStringSlice(m.Properties.GetSrcs(ctx), m.Properties.Build.SourceProps.Specials)
	text += "LOCAL_SRC_FILES := " + strings.Join(srcs, " ") + "\n"

	text += "LOCAL_C_INCLUDES := " + strings.Join(includes, " ") + "\n"
	cflagsList := utils.NewStringSlice(m.Properties.Cflags, m.Properties.Export_cflags)
	_, exportedCflags := m.GetExportedVariables(ctx)
	cflagsList = append(cflagsList, exportedCflags...)
	text += "LOCAL_CFLAGS := " + strings.Join(utils.Filter(cflagsList, moduleCompileFlags), " ") + "\n"
	text += "LOCAL_CPPFLAGS := " + strings.Join(utils.Filter(m.Properties.Cxxflags, moduleCompileFlags), " ") + "\n"
	text += "LOCAL_CONLYFLAGS := " + strings.Join(utils.Filter(m.Properties.Conlyflags, moduleCompileFlags), " ") + "\n"

	// Setup module C/C++ standard if requested. Note that this only affects Android O and later.
	text += specifyCompilerStandard("LOCAL_C_STD", utils.NewStringSlice(cflagsList, m.Properties.Conlyflags))
	text += specifyCompilerStandard("LOCAL_CPP_STD", utils.NewStringSlice(cflagsList, m.Properties.Cxxflags))

	// Check for android libraries in ldlibs, and add to
	// shared, header or static libs instead of ldlibs.
	// This means android will add the appropriate
	// includes and build the right things.
	localAndroidSharedLibs := []string{}
	localAndroidHeaderLibs := []string{}
	localAndroidStaticLibs := []string{}
	if len(m.Properties.Ldlibs) > 0 {
		// The following code is similar to filter, but we are
		// transforming the entries at the same time.
		for _, lib := range m.Properties.Ldlibs {
			if strings.HasPrefix(lib, "android.") {
				localAndroidSharedLibs = append(localAndroidSharedLibs, lib)
				continue
			}
			for _, lib2 := range androidSharedLibs {
				if lib[2:] == lib2[3:] {
					localAndroidSharedLibs = append(localAndroidSharedLibs, lib2)
					break
				}
			}
			for _, lib2 := range androidHeaderLibs {
				if lib[2:] == lib2[3:] {
					localAndroidHeaderLibs = append(localAndroidHeaderLibs, lib2)
					break
				}
			}
			for _, lib2 := range androidStaticLibs {
				if lib[2:] == lib2[3:] {
					localAndroidStaticLibs = append(localAndroidStaticLibs, lib2)
					break
				}
			}
		}
		// Filter out the android libs from ldlibs
		m.Properties.Ldlibs = utils.Filter(m.Properties.Ldlibs, noAndroidLdlibs)
	}
	// Similar processing of export ldlibs to ldlibs.
	if len(m.Properties.Export_ldlibs) > 0 {
		// The following code is similar to filter, but we are
		// transforming the entries at the same time.
		for _, lib := range m.Properties.Export_ldlibs {
			for _, lib2 := range androidSharedLibs {
				if lib[2:] == lib2[3:] {
					localAndroidSharedLibs = append(localAndroidSharedLibs, lib2)
					break
				}
			}
			for _, lib2 := range androidHeaderLibs {
				if lib[2:] == lib2[3:] {
					localAndroidHeaderLibs = append(localAndroidHeaderLibs, lib2)
					break
				}
			}
			for _, lib2 := range androidStaticLibs {
				if lib[2:] == lib2[3:] {
					localAndroidStaticLibs = append(localAndroidStaticLibs, lib2)
					break
				}
			}
		}
	}

	// convert Shared_libs, Resolved_static_libs, Whole_static_libs to
	// Android module names rather than Bob module names
	sharedLibs := []string{}
	staticLibs := []string{}
	wholeStaticLibs := []string{}
	for _, mod := range m.Properties.Shared_libs {
		sharedLibs = append(sharedLibs, androidModuleName(mod))
	}
	for _, mod := range m.Properties.ResolvedStaticLibs {
		staticLibs = append(staticLibs, androidModuleName(mod))
	}
	for _, mod := range m.Properties.Whole_static_libs {
		wholeStaticLibs = append(wholeStaticLibs, androidModuleName(mod))
	}

	text += "LOCAL_SHARED_LIBRARIES := " + strings.Join(localAndroidSharedLibs, " ") + " " +
		strings.Join(sharedLibs, " ") +
		" liblog libc++\n"
	text += "LOCAL_STATIC_LIBRARIES := " + strings.Join(staticLibs, " ") + "\n"
	text += "LOCAL_WHOLE_STATIC_LIBRARIES := " + strings.Join(wholeStaticLibs, " ") + "\n"

	text += "LOCAL_MODULE_TAGS := " + strings.Join(m.Properties.Tags, " ") + "\n"
	text += "LOCAL_EXPORT_C_INCLUDE_DIRS := " + strings.Join(exportIncludeDirs, " ") + "\n"
	if m.Properties.Owner != "" {
		text += "LOCAL_MODULE_OWNER := " + m.Properties.Owner + "\n"
		text += "LOCAL_PROPRIETARY_MODULE := true\n"
	}

	if len(localAndroidHeaderLibs) > 0 {
		text += "LOCAL_HEADER_LIBRARIES := " + strings.Join(localAndroidHeaderLibs, " ") + "\n"
	}

	if len(localAndroidStaticLibs) > 0 {
		text += "LOCAL_STATIC_LIBRARIES += " + strings.Join(localAndroidStaticLibs, " ") + "\n"
	}

	// Can't see a way to wrap a particular library in -Wl in link flags on android, so specify
	// -Wl,--copy-dt-needed-entries across the lot
	hasForwardingLib := false
	copydtneeded := ""
	ctx.VisitDirectDepsIf(
		func(p blueprint.Module) bool { return ctx.OtherModuleDependencyTag(p) == sharedDepTag },
		func(p blueprint.Module) {
			if sl, ok := p.(*sharedLibrary); ok {
				b := sl.build()
				if b.isForwardingSharedLibrary() {
					hasForwardingLib = true
				}
			} else if _, ok := p.(*generateSharedLibrary); ok {
				// Generated forwarding lib not supported
			} else {
				panic(errors.New(ctx.OtherModuleName(p) + " is not a shared library"))
			}
		})
	if hasForwardingLib {
		copydtneeded = "-Wl,--copy-dt-needed-entries"
	}

	// Handle installation
	installGroupPath, ok := getInstallGroupPath(ctx)

	// Only setup multilib for target modules.
	// Normally this should only apply to target libraries, but we
	// also do multilib target binaries to allow creation of test
	// binaries in both modes.
	// All test binaries will be installable.
	isMultiLib := (m.Properties.TargetType == tgtTypeTarget) &&
		((binType == binTypeShared) || (binType == binTypeStatic) || ok)

	if ok {
		text += "LOCAL_MODULE_RELATIVE_PATH:=" + m.Properties.Relative_install_path + "\n"
		if m.Properties.Post_install_cmd != "" {
			// Setup args like we do for bob_generated_*
			args := map[string]string{}
			args["bob_config"] = "$(BOB_ANDROIDMK_DIR)/" + configName
			args["tool"] = filepath.Join("$(LOCAL_PATH)", ctx.ModuleDir(), m.Properties.Post_install_tool)
			args["out"] = "$(LOCAL_INSTALLED_MODULE)"

			// We can't use target specific variables in make due to
			// the way LOCAL_POST_INSTALL_CMD is
			// implemented. Therefore expand all variable use here.
			cmd := m.Properties.Post_install_cmd
			for key, value := range args {
				cmd = strings.Replace(cmd, "${"+key+"}", value, -1)
			}

			// Intentionally using a recursively expanded variable. We
			// don't want LOCAL_INSTALLED_MODULE expanded now, but
			// when it is used in base_rules.mk
			text += "LOCAL_POST_INSTALL_CMD=" + cmd + "\n"
		}

		if binType == binTypeExecutable {
			if isMultiLib {
				// For executables we need to be clear about where to
				// install both 32 and 64 bit versions of the
				// binaries.
				// LOCAL_UNSTRIPPED_PATH does not need to be set
				text += "LOCAL_MODULE_PATH_32:=" + installGroupPath + "\n"
				text += "LOCAL_MODULE_PATH_64:=" + installGroupPath + "64\n"
			} else {
				// When LOCAL_MODULE_PATH is specified, you need to
				// specify LOCAL_UNSTRIPPED_PATH too
				text += "LOCAL_MODULE_PATH:=" + installGroupPath + "\n"

				if m.Properties.TargetType == tgtTypeTarget {
					// Unstripped executables only generated for target
					text += "LOCAL_UNSTRIPPED_PATH:=$(TARGET_OUT_EXECUTABLES_UNSTRIPPED)\n"
				}
			}
		} else {
			// You can't specify an explicit install dir for
			// libraries, you have to use
			// LOCAL_MODULE_RELATIVE_PATH
		}

		requiredModuleNames := m.getInstallDepPhonyNames(ctx)
		if len(requiredModuleNames) > 0 {
			for i, v := range requiredModuleNames {
				requiredModuleNames[i] = androidModuleName(v)
			}
			text += "LOCAL_REQUIRED_MODULES:=" + newlineSeparatedList(requiredModuleNames)
		}
	} else {
		// Only disable installation on the target, because host
		// libraries need to be installed to be used by the build.
		//
		// Target shared libraries do not need an explicit installation
		// location, but cannot be uninstallable, or the multilib paths
		// will conflict, resulting in the same location being used for
		// both 32 and 64-bit versions.
		if m.Properties.TargetType == tgtTypeTarget && binType != binTypeShared {
			text += "LOCAL_UNINSTALLABLE_MODULE:=true\n"
		}
	}

	if isMultiLib {
		text += "LOCAL_MULTILIB:=both\n"
		text += "LOCAL_LDFLAGS_32:=" + strings.Join(utils.Filter(m.Properties.Ldflags, moduleLinkFlags), " ") + copydtneeded + "\n"
	}
	text += "LOCAL_LDFLAGS:=" + strings.Join(utils.Filter(m.Properties.Ldflags, moduleLinkFlags), " ") + copydtneeded + "\n"

	if m.Properties.TargetType == tgtTypeTarget {
		text += "LOCAL_LDLIBS := " + strings.Join(m.Properties.Ldlibs, " ") + "\n"
	} else {
		text += "LOCAL_LDLIBS_$(HOST_OS) := " + strings.Join(m.Properties.Ldlibs, " ") + "\n"
	}
	text += "\ninclude $(" + rulePrefix[m.Properties.TargetType] + ruleSuffix[binType] + ")\n"

	androidMkWriteString(ctx, m.altShortName(), text)
}

func (g *androidMkGenerator) staticActions(m *staticLibrary, ctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		m.GenerateBuildAction(binTypeStatic, ctx)
	}
}

func (g *androidMkGenerator) sharedActions(m *sharedLibrary, ctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		m.GenerateBuildAction(binTypeShared, ctx)
	}
}

func (g *androidMkGenerator) binaryActions(m *binary, ctx blueprint.ModuleContext) {
	if enabledAndRequired(m) {
		m.GenerateBuildAction(binTypeExecutable, ctx)
	}
}

func (*androidMkGenerator) declareAlias(name string, srcs []string) string {
	text := "\ninclude $(CLEAR_VARS)\n\n"
	text += "LOCAL_MODULE := " + name + "\n"

	text += "LOCAL_REQUIRED_MODULES :=" + newlineSeparatedList(srcs)

	text += "\n.PHONY: " + name + "\n"
	text += name + ": $(LOCAL_REQUIRED_MODULES)\n\n"

	text += "include $(base_rules.mk)\n"

	return text
}

func (g *androidMkGenerator) aliasActions(m *alias, ctx blueprint.ModuleContext) {
	text := g.declareAlias(m.Name(), m.Properties.Srcs)
	androidMkWriteString(ctx, m.Name(), text)
}

func pathToModuleName(path string) string {
	path = strings.Replace(path, "/", "__", -1)
	path = strings.Replace(path, ".", "_", -1)
	path = strings.Replace(path, "(", "_", -1)
	path = strings.Replace(path, "$", "_", -1)
	path = strings.Replace(path, ")", "_", -1)
	return path
}

func (g *androidMkGenerator) resourceActions(m *resource, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	installGroupPath, ok := getInstallGroupPath(ctx)
	if !ok {
		androidMkWriteString(ctx, m.altShortName(), "")
		return
	}

	filesToInstall := m.filesToInstall(ctx)
	requiredModuleNames := m.getInstallDepPhonyNames(ctx)

	text := ""

	for _, file := range filesToInstall {
		moduleName := pathToModuleName(file)
		requiredModuleNames = append(requiredModuleNames, moduleName)

		text += "\ninclude $(CLEAR_VARS)\n\n"
		text += "LOCAL_MODULE := " + moduleName + "\n"
		text += "LOCAL_INSTALLED_MODULE_STEM := " + filepath.Base(file) + "\n"
		text += "LOCAL_MODULE_CLASS := ETC\n"
		text += "LOCAL_MODULE_PATH := " + installGroupPath + "\n"
		text += "LOCAL_MODULE_RELATIVE_PATH := " + m.Properties.Relative_install_path + "\n"
		text += "LOCAL_MODULE_TAGS := " + strings.Join(m.Properties.Tags, " ") + "\n"
		text += "LOCAL_SRC_FILES := " + file + "\n"
		text += "\ninclude $(BUILD_PREBUILT)\n"
	}

	text += g.declareAlias(m.Name(), requiredModuleNames)

	androidMkWriteString(ctx, m.altShortName(), text)
}

func (g *androidMkGenerator) sourcePrefix() string {
	return "$(LOCAL_PATH)"
}

func (g *androidMkGenerator) buildDir() string {
	return "$(BOB_ANDROIDMK_DIR)"
}

func outputDirVarName(m *generateCommon) string {
	return m.Name() + "_GEN_DIR"
}

func (g *androidMkGenerator) sourceOutputDir(m *generateCommon) string {
	if m.Properties.Target != tgtTypeHost {
		return filepath.Join("$(TARGET_OUT_GEN)", "STATIC_LIBRARIES", m.Name())
	}
	return filepath.Join("$(HOST_OUT_GEN)", "STATIC_LIBRARIES", m.Name())
}

func outputsVarName(m *generateCommon) string {
	return m.Name() + "_OUTPUTS"
}

func (g *androidMkGenerator) outputs(m *generateCommon) string {
	return "$(" + outputsVarName(m) + ")"
}

// The following makefile snippets are based on Android makefiles from AOSP
//  See aosp/build/core/make/prebuilt_internal.mk
const cmnLibraryMkText string = "include $(BUILD_SYSTEM)/base_rules.mk\n\n" +
	"export_includes:=$(intermediates)/export_includes\n" +

	//  Setup rule to create export_includes
	"$(export_includes): PRIVATE_EXPORT_C_INCLUDE_DIRS:=$(LOCAL_EXPORT_C_INCLUDE_DIRS)\n" +
	"$(export_includes): $(LOCAL_MODULE_MAKEFILE_DEP)\n" +
	"\t@echo Export includes file: $< -- $@\n" +
	"\t$(hide) mkdir -p $(dir $@) && rm -f $@\n" +
	"ifdef LOCAL_EXPORT_C_INCLUDE_DIRS\n" +
	"\t$(hide) for d in $(PRIVATE_EXPORT_C_INCLUDE_DIRS); do \\\n" +
	"\t\techo \"-I $$d\" >> $@; \\\n" +
	"\t\tdone\n" +
	"else\n" +
	"\t$(hide) touch $@\n" +
	"endif\n\n" +

	"$(LOCAL_BUILT_MODULE): $(LOCAL_SRC_FILES) | $(export_includes)\n" +
	"\tmkdir -p $(dir $@)\n" +
	"\tcp $< $@\n\n" +

	// Setup link type
	// We assume LOCAL_SDK_VERSION and LOCAL_USE_VNDK will not be set
	"ifeq ($(PLATFORM_SDK_VERSION),25)\n" +
	"  # link_type not required\n" +

	// Android O only.
	"else ifeq ($(PLATFORM_SDK_VERSION),26)\n" +
	"  my_link_type := $(intermediates)/link_type\n\n" +

	"$(my_link_type): PRIVATE_LINK_TYPE := native:platform\n" +
	"$(eval $(call link-type-partitions,$(my_link_type)))\n" +
	"$(my_link_type):\n" +
	"\t@echo Check module type: $@\n" +
	"\t$(hide) mkdir -p $(dir $@) && rm -f $@\n" +
	"\t$(hide) echo \"$(PRIVATE_LINK_TYPE)\" >$@\n" +

	"$(LOCAL_BUILT_MODULE): | $(my_link_type)\n\n" +

	// Android OMR1 and later
	"else\n" +
	"  include $(BUILD_SYSTEM)/allowed_ndk_types.mk\n\n" +
	"  my_link_type := native:platform\n" +
	"  my_link_deps :=\n" +
	"  my_2nd_arch_prefix := $(LOCAL_2ND_ARCH_VAR_PREFIX)\n" +
	"  my_common :=\n" +
	"  include $(BUILD_SYSTEM)/link_type.mk\n" +
	"endif\n"

func declarePrebuiltStaticLib(moduleName, path, includePaths string, target bool) string {
	text := "\ninclude $(CLEAR_VARS)\n"
	text += "LOCAL_MODULE:=" + moduleName + "\n"
	text += "LOCAL_SRC_FILES:=" + path + "\n"
	if !target {
		text += "LOCAL_IS_HOST_MODULE:=true\n"
	}

	// We would like to just have the following line, but it looks like it is NDK only
	// Therefore all the following is needed.
	//text += "include $(PREBUILT_STATIC_LIBRARY)\n"

	text += "LOCAL_MODULE_CLASS:=STATIC_LIBRARIES\n"
	text += "LOCAL_UNINSTALLABLE_MODULE:=true\n"
	text += "LOCAL_MODULE_SUFFIX:=.a\n"

	if includePaths != "" {
		text += "LOCAL_EXPORT_C_INCLUDE_DIRS:=" + includePaths + "\n"
	}

	text += cmnLibraryMkText
	return text
}

func declarePrebuiltSharedLib(moduleName, path, includePaths string, target bool) string {
	text := "\ninclude $(CLEAR_VARS)\n"
	text += "LOCAL_MODULE:=" + moduleName + "\n"
	text += "LOCAL_SRC_FILES:=" + path + "\n"
	if !target {
		text += "LOCAL_IS_HOST_MODULE:=true\n"
	}
	// We would like to just have the following line, but it looks like it is NDK only
	// Therefore all the following is needed.
	//text += "include $(PREBUILT_SHARED_LIBRARY)\n"

	text += "LOCAL_MODULE_CLASS:=SHARED_LIBRARIES\n"
	text += "LOCAL_MODULE_SUFFIX:=.so\n"

	if includePaths != "" {
		text += "LOCAL_EXPORT_C_INCLUDE_DIRS:=" + includePaths + "\n"
	}

	//  Put shared libraries in common path to simplify link line.
	//  see shared_library_internal.mk and host_shared_library_internal.mk
	if target {
		// Only supporting the primary target architecture.
		// Others are TARGET_2ND_x
		text += "OVERRIDE_BUILT_MODULE_PATH:=$(TARGET_OUT_INTERMEDIATE_LIBRARIES)\n\n"
	} else {
		// Only supporting the primary host architecture.
		// Others are HOST_2ND_x, HOST_CROSS_x, HOST_CROSS_2ND_x
		text += "OVERRIDE_BUILT_MODULE_PATH:=$(HOST_OUT_INTERMEDIATE_LIBRARIES)\n\n"
	}

	text += cmnLibraryMkText
	return text
}

func declarePrebuiltBinary(moduleName, path string, target bool) string {
	text := "\ninclude $(CLEAR_VARS)\n"
	text += "LOCAL_MODULE:=" + moduleName + "\n"
	text += "LOCAL_SRC_FILES:=" + path + "\n"
	if !target {
		text += "LOCAL_IS_HOST_MODULE:=true\n"
	}

	text += "LOCAL_MODULE_CLASS:=EXECUTABLES\n"
	text += "LOCAL_MODULE_SUFFIX:=\n\n"

	text += "include $(BUILD_SYSTEM)/base_rules.mk\n\n"

	text += "$(LOCAL_BUILT_MODULE): $(LOCAL_SRC_FILES)\n"
	text += "\tmkdir -p $(dir $@)\n"
	text += "\tcp $< $@\n"

	return text
}

func installGeneratedFiles(m installable, ctx blueprint.ModuleContext, tags []string) string {
	/* Install generated files one by one, if required */
	installGroupPath, ok := getInstallGroupPath(ctx)

	if !ok {
		return ""
	}

	text := "\n"
	filesToInstall := m.filesToInstall(ctx)

	for _, file := range filesToInstall {
		moduleName := pathToModuleName(file)

		text += "include $(CLEAR_VARS)\n\n"

		text += "LOCAL_MODULE := " + moduleName + "\n"
		text += "LOCAL_INSTALLED_MODULE_STEM := " + filepath.Base(file) + "\n"
		text += "LOCAL_MODULE_CLASS := ETC\n"
		text += "LOCAL_MODULE_PATH := " + installGroupPath + "\n"
		text += "LOCAL_MODULE_RELATIVE_PATH := " + m.getInstallableProps().Relative_install_path + "\n"
		text += "LOCAL_MODULE_TAGS := " + strings.Join(tags, " ") + "\n"
		text += "LOCAL_PREBUILT_MODULE_FILE := " + file + "\n\n"

		text += "include $(BUILD_PREBUILT)\n"
	}

	return text
}

func (g *androidMkGenerator) generateCommonActions(m *generateCommon, ctx blueprint.ModuleContext, inouts []inout) string {
	text := "##########################\ninclude $(CLEAR_VARS)\n\n"

	// This is required to have $(local-generated-sources-dir) work as expected
	text += "LOCAL_MODULE := " + m.Name() + "\n"
	text += "LOCAL_MODULE_CLASS := STATIC_LIBRARIES\n"
	text += outputsVarName(m) + " := \n"
	text += outputDirVarName(m) + " := " + g.sourceOutputDir(m) + "\n"
	text += "\n"

	cmd, args, implicits, _ := m.getArgs(ctx)
	utils.StripUnusedArgs(args, cmd)

	for _, inout := range inouts {
		if _, ok := args["headers_generated"]; ok {
			headers := utils.Filter(inout.out, utils.IsHeader)
			args["header_generated"] = strings.Join(headers, " ")
		}
		if _, ok := args["srcs_generated"]; ok {
			sources := utils.Filter(inout.out, utils.IsSource)
			args["srcs_generated"] = strings.Join(sources, " ")
		}

		ins := utils.Join(utils.PrefixDirs(inout.srcIn, g.sourcePrefix()), inout.genIn)

		// Make does not cleanly support multiple out-files
		// To handle that, we output the rule only on the first file, and let every other output
		// depend on the first.
		// This is not 100 % safe, since if the secondary file is removed, it will not be rebuilt.
		// It is assumed that this will not be a big issue, since removing individual files from a generated
		// directory should not be common.
		outs := inout.out[0]
		for _, key := range utils.SortedKeys(args) {
			text += outs + ": " + key + ":= " + args[key] + "\n"
		}

		text += outs + ": in := " + ins + "\n"
		text += outs + ": out := " + strings.Join(inout.out, " ") + "\n"
		text += outs + ": depfile := " + inout.depfile + "\n"
		text += outs + ": " + ins + " " + strings.Join(inout.implicitSrcs, " ") + "\n"
		text += "\t" + cmd + "\n"
		if inout.depfile != "" {
			// Convert the depfile file format as part of this rule.
			text += g.transformDepFile("$(depfile)")
			// ...and include it outside the rule.
			text += g.includeDepFile(outs, inout.depfile)
		}
		text += outputsVarName(m) + " += " + outs + "\n"

		for _, out := range inout.out[1:] {
			text += out + ": " + outs + "\n"
			text += outputsVarName(m) + " += " + out + "\n"
		}
		text += "\n"
	}
	text += g.outputs(m) + ": " + strings.Join(implicits, " ") + "\n"

	/* This will ensure that any dependencies will not be rebuilt in the case of no change */
	text += ".KATI_RESTAT: " + g.outputs(m) + "\n"

	return text
}

func (g *androidMkGenerator) generateSourceActions(m *generateSource, ctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		text := g.generateCommonActions(&m.generateCommon, ctx, inouts)
		text += installGeneratedFiles(m, ctx, m.generateCommon.Properties.Tags)
		androidMkWriteString(ctx, m.altShortName(), text)
	}
}

func (g *androidMkGenerator) transformSourceActions(m *transformSource, ctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		text := g.generateCommonActions(&m.generateCommon, ctx, inouts)
		text += installGeneratedFiles(m, ctx, m.generateCommon.Properties.Tags)
		androidMkWriteString(ctx, m.altShortName(), text)
	}
}

func (g *androidMkGenerator) genStaticActions(m *generateStaticLibrary, ctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		text := g.generateCommonActions(&m.generateCommon, ctx, inouts)

		// Add prebuilt outputs
		includeDirs := utils.PrefixDirs(m.generateCommon.Properties.Export_gen_include_dirs, g.sourceOutputDir(&m.generateCommon))
		text += declarePrebuiltStaticLib(m.altShortName(), getLibraryGeneratedPath(m, g),
			strings.Join(includeDirs, " "),
			m.generateCommon.Properties.Target != tgtTypeHost)

		androidMkWriteString(ctx, m.altShortName(), text)
	}
}

func (g *androidMkGenerator) genSharedActions(m *generateSharedLibrary, ctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		text := g.generateCommonActions(&m.generateCommon, ctx, inouts)

		// Add prebuilt outputs
		includeDirs := utils.PrefixDirs(m.generateCommon.Properties.Export_gen_include_dirs, g.sourceOutputDir(&m.generateCommon))
		text += declarePrebuiltSharedLib(m.altShortName(), getLibraryGeneratedPath(m, g),
			strings.Join(includeDirs, " "),
			m.generateCommon.Properties.Target != tgtTypeHost)

		androidMkWriteString(ctx, m.altShortName(), text)
	}
}

func (g *androidMkGenerator) genBinaryActions(m *generateBinary, ctx blueprint.ModuleContext, inouts []inout) {
	if enabledAndRequired(m) {
		text := g.generateCommonActions(&m.generateCommon, ctx, inouts)

		// Add prebuilt outputs
		text += declarePrebuiltBinary(m.altShortName(), getLibraryGeneratedPath(m, g),
			m.generateCommon.Properties.Target != tgtTypeHost)

		androidMkWriteString(ctx, m.altShortName(), text)
	}
}

func (g *androidMkGenerator) binaryOutputDir(m *binary) string {
	return "$(HOST_OUT_EXECUTABLES)"
}

type androidMkOrderer struct {
}

type androidMkFile struct {
	Name string
	Deps []string
}

type androidMkFileSlice []androidMkFile

type androidNaming interface {
	// Alternate name for module output, also used in preference to the
	// module name for the Android module name
	altName() string
	// Alternate brief name for the module
	altShortName() string
}

func enabledAndRequired(m blueprint.Module) bool {
	if e, ok := m.(enableable); ok {
		if !isEnabled(e) || !isRequired(e) {
			return false
		}
	}
	return true
}

func (s *androidMkOrderer) GenerateBuildActions(ctx blueprint.SingletonContext) {

	var order androidMkFileSlice
	ctx.VisitAllModules(func(m blueprint.Module) {
		di, ok := m.(androidNaming)
		if ok && enabledAndRequired(m) {
			deps := []string{}
			ctx.VisitDepsDepthFirst(m, func(child blueprint.Module) {
				childdi, ok := child.(androidNaming)
				_, isDefaults := child.(*defaults)
				if ok && !isDefaults && enabledAndRequired(m) {
					deps = append(deps, childdi.altShortName())
				}
			})
			_, isDefaults := di.(*defaults)
			if !isDefaults {
				order = append(order, androidMkFile{di.altShortName(), deps})
			}
		}
	})

	text := ""
	// Quick and dirty alphabetical topological sort
	for len(order) > 0 {
		lowindex := -1
		for i, val := range order {
			if (lowindex == -1 || val.Name < order[lowindex].Name) && len(val.Deps) == 0 {
				lowindex = i
			}
		}
		if lowindex == -1 {

			/* Generate a list of remaining modules and their dependencies */
			deps := ""
			for _, o := range order {
				deps += fmt.Sprintf("%s depends on\n", o.Name)
				for _, d := range o.Deps {
					deps += fmt.Sprintf("\t%s\n", d)
				}
			}

			panic(fmt.Errorf("unmet or circular dependency. %d remaining.\n%s", len(order), deps))
		}
		text += "include $(BOB_ANDROIDMK_DIR)/" + order[lowindex].Name + ".inc\n"
		for i := range order {
			newdeps := []string{}
			for _, dep := range order[i].Deps {
				if dep != order[lowindex].Name {
					newdeps = append(newdeps, dep)
				}
			}
			order[i].Deps = newdeps
		}
		order = append(order[:lowindex], order[lowindex+1:]...)
	}
	writeIfChanged(filepath.Join(builddir, "Android.inc"), text)
}

func (g *androidMkGenerator) moduleOutputDir(moduleName string) string {
	return fmt.Sprintf("$(dir $(ALL_MODULES.%s.BUILT))", moduleName)
}

func (g *androidMkGenerator) kernelModOutputDir(m *kernelModule) string {
	return g.moduleOutputDir(m.altName())
}

func (g *androidMkGenerator) staticLibOutputDir(m *staticLibrary) string {
	return g.moduleOutputDir(m.altName())
}

func (g *androidMkGenerator) sharedLibOutputDir(m *sharedLibrary) string {
	return g.moduleOutputDir(m.altName())
}

func (g *androidMkGenerator) sharedLibsDir(tgtType string) string {
	if tgtType != tgtTypeHost {
		return "$(TARGET_OUT_SHARED_LIBRARIES)"
	}
	return "$(HOST_OUT_SHARED_LIBRARIES)"
}

func (g *androidMkGenerator) transformDepFile(depfile string) (text string) {
	text += "ifeq ($(word 1, $(subst ., ,$(PLATFORM_VERSION))),7)\n"
	text += "\t$(call transform-d-to-p-args," + depfile + "," + depfile + ".P)\n"
	text += "endif\n"
	return
}

func (g *androidMkGenerator) includeDepFile(target string, depfile string) (text string) {
	text += "ifeq ($(word 1, $(subst ., ,$(PLATFORM_VERSION))),7)\n"
	text += "  $(call include-depfile," + depfile + ".P," + target + ")\n"
	text += "else\n"
	text += "  $(call include-depfile," + depfile + "," + target + ")\n"
	text += "endif\n"
	return
}

func (g *androidMkGenerator) kernelModuleActions(m *kernelModule, ctx blueprint.ModuleContext) {
	if !enabledAndRequired(m) {
		return
	}

	text := "##########################\ninclude $(CLEAR_VARS)\n\n"
	text += "LOCAL_MODULE := " + m.altShortName() + "\n"
	text += "LOCAL_MODULE_CLASS := KERNEL_MODULES\n"
	text += "LOCAL_CLANG := false\n"
	text += "LOCAL_MODULE_TAGS := " + strings.Join(m.Properties.Tags, " ") + "\n\n"

	sources := m.Properties.GetSrcs(ctx)

	text += "LOCAL_SRC_FILES :=" + newlineSeparatedList(sources)

	// Now the build rules (i.e. what would be done by an 'include
	// $(BUILD_KERNEL_MODULE)', if there was one).
	text += "TARGET_OUT_$(LOCAL_MODULE_CLASS) := $(TARGET_OUT)/lib/modules\n"
	installGroupPath, ok := getInstallGroupPath(ctx)
	if ok {
		text += "LOCAL_MODULE_PATH := " + installGroupPath + "\n"
		text += "LOCAL_MODULE_RELATIVE_PATH := " + m.Properties.Relative_install_path + "\n"
	} else {
		text += "LOCAL_UNINSTALLABLE_MODULE := true\n"
	}
	text += "LOCAL_MODULE_SUFFIX := .ko\n"
	if m.Properties.Owner != "" {
		text += "LOCAL_MODULE_OWNER := " + m.Properties.Owner + "\n"
		text += "LOCAL_PROPRIETARY_MODULE := true\n"
	}
	text += "include $(BUILD_SYSTEM)/base_rules.mk\n\n"

	args := m.generateKbuildArgs(ctx)
	args["sources"] = "$(addprefix $(LOCAL_PATH)/,$(LOCAL_SRC_FILES))"
	args["kmod_build"] = "$(LOCAL_PATH)/" + args["kmod_build"]
	args["local_path"] = "$(LOCAL_PATH)"

	// Create a target-specific variable declaration for each required parameter.
	for _, key := range utils.SortedKeys(args) {
		text += fmt.Sprintf("$(LOCAL_BUILT_MODULE): %s := %s\n", key, args[key])
	}

	text += "\n$(LOCAL_BUILT_MODULE): $(LOCAL_MODULE_MAKEFILE_DEP) " +
		args["sources"] + " " +
		args["kmod_build"] + " " +
		strings.Join(m.extraSymbolsFiles(ctx), " ") + "\n"
	text += "\tmkdir -p \"$(@D)\"\n"
	cmd := "python $(kmod_build) --output $@ --depfile $@.d " +
		"--common-root $(local_path) " +
		"--module-dir \"$(output_module_dir)\" $(extra_includes) " +
		"--sources $(sources) $(kbuild_extra_symbols) " +
		"--kernel \"$(kernel_dir)\" --cross-compile \"$(kernel_compiler)\" " +
		"$(kbuild_options) --extra-cflags \"$(extra_cflags)\" $(make_args)"

	text += "\techo " + cmd + "\n"
	text += "\t" + cmd + "\n"
	text += g.transformDepFile("$@.d") + "\n"

	text += g.includeDepFile("$(LOCAL_BUILT_MODULE)", "$(LOCAL_BUILT_MODULE).d")

	// The Module.symvers file is generated during Kbuild, but make doesn't
	// support multiple output files in a rule, so add it as a dependency
	// of the module.
	text += fmt.Sprintf("\n$(dir $(LOCAL_BUILT_MODULE))/Module.symvers: $(LOCAL_BUILT_MODULE)\n")

	androidMkWriteString(ctx, m.altShortName(), text)
}

func androidMkOrdererFactory() blueprint.Singleton {
	return &androidMkOrderer{}
}

var (
	androidModuleNameMap    = map[string]string{}
	androidModuleReverseMap = map[string]string{}
	androidModuleMapLock    sync.RWMutex
)

func androidModuleName(name string) string {
	androidModuleMapLock.RLock()
	defer androidModuleMapLock.RUnlock()
	return androidModuleNameMap[name]
}

func mapAndroidNames(ctx blueprint.BottomUpMutatorContext) {
	if m, ok := ctx.Module().(androidNaming); ok {
		// Ignore defaults
		if _, ok := ctx.Module().(*defaults); ok {
			return
		}

		if enabledAndRequired(ctx.Module()) {
			androidModuleMapLock.Lock()
			defer androidModuleMapLock.Unlock()

			if existing, ok := androidModuleReverseMap[m.altName()]; ok {
				if existing != ctx.ModuleName() {
					panic(fmt.Errorf("out name collision. Both %s and %s are required and map to %s",
						ctx.ModuleName(), existing, m.altName()))
				}
			}
			androidModuleNameMap[ctx.ModuleName()] = m.altName()
			androidModuleReverseMap[m.altName()] = ctx.ModuleName()
		}
	}
}

func (g *androidMkGenerator) init(ctx *blueprint.Context, config *bobConfig) {
	ctx.RegisterBottomUpMutator("modulemapper", mapAndroidNames).Parallel()

	ctx.RegisterSingletonType("androidmk_orderer", androidMkOrdererFactory)

	g.toolchainSet.parseConfig(config)
}
