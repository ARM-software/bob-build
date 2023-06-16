/*
 * Copyright 2023 Arm Limited.
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
	"strings"

	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

// BuildProps contains properties required by all modules that compile C/C++
type BuildProps struct {
	// Alternate output name, used for the file name and Android rules
	Out *string
	// Flags exported for dependent modules
	Export_cflags []string
	// Flags used for C compilation
	Conlyflags []string
	// Flags used for C++ compilation
	Cxxflags []string
	// Flags used for assembly compilation
	Asflags []string
	// Flags used for linking
	Ldflags []string
	// Same as ldflags, but specified on static libraries and propagated to
	// the top-level build object.
	Export_ldflags []string
	// Shared library version
	Library_version string
	// Shared library version script
	Version_script *string

	// The list of shared lib modules that this library depends on.
	// These are propagated to the closest linking object when specified on static libraries.
	// shared_libs is an indication that this module is using a shared library, and
	// users of this module need to link against it.
	Shared_libs []string `bob:"first_overrides"`
	// The libraries mentioned here will be appended to shared_libs of the modules that use
	// this library (via static_libs, whole_static_libs or shared_libs).
	ExtraSharedLibs []string `blueprint:"mutated"`

	// The list of static lib modules that this library depends on
	// These are propagated to the closest linking object when specified on static libraries.
	// static_libs is an indication that this module is using a static library, and
	// users of this module need to link against it.
	Static_libs []string `bob:"first_overrides"`

	// This list of dependencies that exported cflags and exported include dirs
	// should be propagated 1-level higher
	Reexport_libs []string `bob:"first_overrides"`
	// Internal property for collecting libraries with reexported flags and include paths
	ResolvedReexportedLibs []string `blueprint:"mutated"`

	ResolvedStaticLibs []string `blueprint:"mutated"`

	// The list of whole static libraries that this library depnds on
	// This will include all the objects in the library (as opposed to normal static linking)
	// If this is set for a static library, any shared library will also include objects
	// from dependent libraries
	Whole_static_libs []string `bob:"first_overrides"`

	// List of libraries to import headers from, but not link to
	Header_libs []string `bob:"first_overrides"`

	// List of libraries that users of the current library should import
	// headers from, but not link to
	Export_header_libs []string `bob:"first_overrides"`

	// Linker flags required to link to the necessary system libraries
	// These are propagated to the closest linking object when specified on static libraries.
	Ldlibs []string `bob:"first_overrides"`

	// The list of modules that generate extra headers for this module
	Generated_headers []string `bob:"first_overrides"`

	// The list of modules that generate extra headers for this module,
	// which should be made available to linking modules
	Export_generated_headers []string `bob:"first_overrides"`

	// The list of modules that generate extra source files for this module
	Generated_sources []string

	// The list of modules that generate output required by the build wrapper
	Generated_deps []string

	// Include local dirs to be exported into dependent.
	// The system variant will propagate includes using `-isystem`, but use `-I` for
	// current module.
	Export_local_include_dirs        []string `bob:"first_overrides"`
	Export_local_system_include_dirs []string `bob:"first_overrides"`

	// Include dirs (path relative to root) to be exported into dependent.
	// The system variant will propagate includes using `-isystem`, but use `-I` for
	// current module.
	Export_include_dirs        []string `bob:"first_overrides"`
	Export_system_include_dirs []string `bob:"first_overrides"`

	// Wrapper for all build commands (object file compilation *and* linking)
	Build_wrapper *string

	// Adds DT_RPATH symbol to binaries and shared libraries so that they can find
	// their dependencies at runtime.
	Add_lib_dirs_to_rpath *bool

	// This is a shared library that pulls in one or more shared
	// libraries to resolve symbols that the binary needs. This is
	// useful where a named library is the standard library to link
	// against, but the implementation may exist in another
	// library.
	//
	// Only valid on bob_shared_library.
	//
	// Currently we need to link with -Wl,--copy-dt-needed-entries.
	// This makes the binary depend on the implementation library, and
	// requires the BFD linker.
	Forwarding_shlib *bool

	StripProps
	AndroidPGOProps
	AndroidMTEProps

	Hwasan_enabled *bool

	TargetType toolchain.TgtType `blueprint:"mutated"`
}

func (b *BuildProps) processBuildWrapper(ctx blueprint.BaseModuleContext) {
	if b.Build_wrapper != nil {
		// The build wrapper may be a local tool, in which case we
		// need to prefix it with ${SrcDir}. It can also be a tool in
		// PATH like ccache.
		//
		// We want to avoid doing this repeatedly, so try do it in an
		// early mutator
		*b.Build_wrapper = strings.TrimSpace(*b.Build_wrapper)
		firstWord := strings.SplitN(*b.Build_wrapper, " ", 1)[0]

		// If the first character is '/' this is an absolute path, so no need to do anything
		if firstWord[0] != '/' {
			// Otherwise if the first word contains '/' this is a local path
			if strings.ContainsAny(firstWord, "/") {
				*b.Build_wrapper = getBackendPathInSourceDir(getGenerator(ctx), *b.Build_wrapper)
			}
		}
	}
}

// Add module paths to srcs, exclude_srcs, local_include_dirs, export_local_include_dirs
// and post_install_tool
func (b *BuildProps) processPaths(ctx blueprint.BaseModuleContext) {
	prefix := projectModuleDir(ctx)

	b.Export_local_include_dirs = utils.PrefixDirs(b.Export_local_include_dirs, prefix)
	b.Export_local_system_include_dirs = utils.PrefixDirs(b.Export_local_system_include_dirs, prefix)

	b.processBuildWrapper(ctx)
}
