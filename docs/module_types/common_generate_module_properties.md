Common properties for generate modules
======================================

There are two module types supporting source generation.
The first, `bob_generate_source` runs an arbitrary command
once, which processes all of its sources, and generates one
or more output files. The second, `bob_transform_source`, will
run the command once per source file instead, generating at least
one output files per input.

Henceforth, the common aspects of both modules will
be referred to as `bob_generated`.

----
### **bob_generated.cmd** (required)
The command that is to be run for this module. Bob supports various
substitutions in the command, by using `${name_of_var}`. The
available substitutions are:

- `${in}` - space-delimited list of source (input) paths
- `${out}` - space-delimited list of target (output) paths
- `${depfile}` - the path for the generated dependency file
- `${rspfile}` - the path to the RSP file, if `rsp_content` is set
- `${args}` - the value of `args` - space-delimited
- `${tool}` - the path to the script specified by `tool`
- `${host_bin}` - the path to the binary specified by `host_bin`
- `${module_dir}` - the path this module's source directory
- `${gen_dir}` - the path to the output directory for this module
- `${(name)_dir}` - the output directory for the `module_deps` dependency with `name`
- `${(name)_out}` - the outputs of the `module_deps` dependency with `name`
- `${src_dir}` - the path to the project source directory - this will be different
  than the build source directory for Android.

The value in `cmd` is executed by the shell. Compound shell
expressions and expansions can be used, though we recommend keeping
commands simple. If double quotes (") need to be on the shell command
line, they should be escaped with backslash (\) to get through the
blueprint parser. Where a `$` needs to be evaluated by the shell (for
example to expand an environment variable) use `$$`.

The [`match_srcs`](../strings.md#match_srcs) function can be used in
this property to reference files listed in `srcs`.

----
### **bob_generated.tool** (required)
A path to the tool that is to be used in `cmd`. If `${tool}` is in
the command variable, then this will be replaced with the path to
this tool.

----
### **bob_generated.host_bin** (optional)
Refers to a `bob_binary.name` with `host_supported: true` which is used in this
module's command. Specifying this in `host_bin` ensures that the host tool will
be built before the `bob_generated`.

----
### **bob_generated.module_deps** (optional)
A list of other modules that this generator depends on. The dependencies can be
used in the command through `${(name_of_dependency)_dir}` (that is, the variable's
name is the name of the dependency, with the `_dir` suffix).

----
### **bob_generated.module_srcs** (optional)
A list of other modules that this generator depends on.
The dependencies will be added to the list of srcs.

----
### **bob_generated.args** (optional)
A list of `args` that will be space separated and added to the `cmd`.

The [`match_srcs`](../strings.md#match_srcs) function can be used in
this property to reference files listed in `srcs`.

----
### **bob_generated.console** (optional)
This will use Ninja's [console pool](https://ninja-build.org/manual.html#_the_literal_console_literal_pool)
When `true` one job will run at a time - they won't be concurrent.

----
### **bob_generated.export_gen_include_dirs** (optional)
Additional include paths to add for modules that use `generated_headers`. This
will be defined relative to the module-specific build directory.

----
### **bob_generated.encapsulates** (optional)
A list of source modules that this bob_generated_source will encapsulate.
When this module is used with generated_headers, the named modules' export_gen_include_dirs will be forwarded.
When this module is used with generated_sources, the named modules' outputs will be supplied as sources.

----
### **bob_generated.flag_defaults** (optional)
Generated sources may wish to access the build flags being used for "normal"
library or executable modules. `flag_defaults` should contain the name of a
`bob_defaults` module, whose flags will be accessible from this one, by
allowing extra variables to be used in `bob_generated.cmd`: `ar`, `cc`, `cxx`,
`asflags`, `cflags`, `conlyflags`, `cxxflags`, `ldflags` and `ldlibs`.

----
### **bob_generated.target** (required)
The target type - must be either `host` or `target`. This is to choose between
the host and target variant of the `bob_defaults` specified in
`bob_generate.flag_defaults`.

----
### **bob_generated.depfile** (optional)
If true, a dependency file describing discovered dependencies will be generated
with a specific name, derived from module name (`bob_generate_source`) or
source file name (`bob_transform_source`).

----
### **bob_generated.rsp_content** (optional)
If set, the value provided will be expanded and written to a file immediately
before command execution, and the file name will be made available to the
command as `${rspfile}`. This allows commands to use argument lists greater
than the command line length limit, by writing e.g. the input or output list to
a file.
