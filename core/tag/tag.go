package tag

import (
	"github.com/google/blueprint"
)

var (
	AliasTag                  = DependencyTag{Name: "alias"}
	DebugInfoTag              = DependencyTag{Name: "debug_info"}
	DefaultTag                = DependencyTag{Name: "default"}
	ExportGeneratedHeadersTag = DependencyTag{Name: "export_generated_headers"}
	FilegroupTag              = DependencyTag{Name: "filegroup"}
	GeneratedHeadersTag       = DependencyTag{Name: "generated_headers"}
	GeneratedSourcesTag       = DependencyTag{Name: "generated_sources"}
	GeneratedTag              = DependencyTag{Name: "generated_dep"}
	HeaderTag                 = DependencyTag{Name: "header"}
	HostToolBinaryTag         = DependencyTag{Name: "host_tool_bin"}
	ImplicitSourcesTag        = DependencyTag{Name: "implicit_srcs"}
	InstallGroupTag           = DependencyTag{Name: "install_group"}
	InstallTag                = DependencyTag{Name: "install_dep"}
	KernelModuleTag           = DependencyTag{Name: "kernel_module"}
	ReexportLibraryTag        = DependencyTag{Name: "reexport_libs"}
	SharedTag                 = DependencyTag{Name: "shared"}
	StaticTag                 = DependencyTag{Name: "static"}
	ToolchainTag              = DependencyTag{Name: "toolchain"}
	WholeStaticTag            = DependencyTag{Name: "whole_static"}
	DepTag                    = DependencyTag{Name: "dep"} // Generic deps used by new targets.
)

// DependencyTag contains the name of the tag used to track a particular type
// of dependency between modules
type DependencyTag struct {
	blueprint.BaseDependencyTag
	Name string
}
