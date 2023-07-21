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
	DepTag                    = DependencyTag{name: "dep"} // Generic deps used by new targets.
)

// DependencyTag contains the name of the tag used to track a particular type
// of dependency between modules
type DependencyTag struct {
	blueprint.BaseDependencyTag
	name string
}
