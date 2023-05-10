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
	"github.com/google/blueprint"
)

var (
	AliasTag                  = DependencyTag{name: "alias"}
	DebugInfoTag              = DependencyTag{name: "debug_info"}
	DefaultTag                = DependencyTag{name: "default"}
	ExportGeneratedHeadersTag = DependencyTag{name: "export_generated_headers"}
	FilegroupTag              = DependencyTag{name: "filegroup"}
	GeneratedHeadersTag       = DependencyTag{name: "generated_headers"}
	GeneratedSourcesTag       = DependencyTag{name: "generated_sources"}
	GeneratedTag              = DependencyTag{name: "generated_dep"}
	HeaderTag                 = DependencyTag{name: "header"}
	HostToolBinaryTag         = DependencyTag{name: "host_tool_bin"}
	ImplicitSourcesTag        = DependencyTag{name: "implicit_srcs"}
	InstallGroupTag           = DependencyTag{name: "install_group"}
	InstallTag                = DependencyTag{name: "install_dep"}
	KernelModuleTag           = DependencyTag{name: "kernel_module"}
	ReexportLibraryTag        = DependencyTag{name: "reexport_libs"}
	SharedTag                 = DependencyTag{name: "shared"}
	StaticTag                 = DependencyTag{name: "static"}
	WholeStaticTag            = DependencyTag{name: "whole_static"}
)

// DependencyTag contains the name of the tag used to track a particular type
// of dependency between modules
type DependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}
