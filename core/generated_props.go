package core

import "github.com/ARM-software/bob-build/core/toolchain"

// GenerateProps contains the module properties that allow generation of
// output from arbitrary commands
type GenerateProps struct {
	LegacySourceProps
	AliasableProps
	EnableableProps
	InstallableProps

	/* The command that is to be run for this source generation.
	 * Substitutions can be made in the command, by using $name_of_var. A list of substitutions that can be used:
	 * $gen_dir      - the path to the directory which belongs to this source generator
	 * $in           - the path to the sources - space-delimited
	 * $out          - the path to the targets - space-delimited
	 * $depfile      - the path to generated dependency file
	 * $args         - the value of "args" - space-delimited
	 * $tool         - the path to the tool
	 * $tool <label> - the path to the tool with name <label>
	 * $host_bin     - the path to the binary that is produced by the host_bin module
	 * $(dep)_out    - the outputs of the generated_dep `dep`
	 * $src_dir      - the path to the project source directory - this will be different than the build source directory
	 *                 for Android.
	 * $module_dir   - the path to the module directory */
	Cmd *string

	// A paths to the tool that are to be used in cmd. If $tool is in the command variable, then this will be replaced
	// with the path to this tool. ${tool} refers to the first tool in a list. To reference
	// other tool use index syntax ${tool <label>} (e.g. ${tool fixer.py} for `fixer.py` tool from list).
	Tools []string

	// Adds a dependency on a binary with `host_supported: true` which is used by this module.
	// The path can be referenced in cmd as ${host_bin}.
	Host_bin *string

	// Values to use on Android for LOCAL_MODULE_TAGS, defining which builds this module is built for
	// TODO: Hide this in Android-specific properties
	Tags []string

	// A list of other modules that this generator depends on. The dependencies can be used in the command through
	// $name_of_dependency_dir .
	Generated_deps []string

	// A list of other modules that this generator depends on. The dependencies will be add to the list of srcs
	Generated_sources []string

	// A list of args that will be spaceseparated and add to the cmd
	Args []string

	// Used to indicate that the console should be used.
	Console *bool

	// A list of source modules that this bob_generated_source will encapsulate.
	// When this module is used with generated_headers, the named modules' export_gen_include_dirs will be forwarded.
	// When this module is used with generated_sources, the named modules' outputs will be supplied as sources.
	Encapsulates []string

	// Additional include paths to add for modules that use generate_headers.
	// This will be defined relative to the module-specific build directory
	Export_gen_include_dirs []string

	// The defaults used to retrieve cflags
	Flag_defaults []string

	// The target type - must be either "host" or "target"
	Target toolchain.TgtType

	// If true, depfile name will be generated and can be used as ${depfile} reference in 'cmd'
	Depfile *bool

	// If set, Ninja will expand the string and write it to a file just
	// before executing the command. This can be used to e.g. contain ${in},
	// in cases where the command line length is a limiting factor.
	Rsp_content *string
}
