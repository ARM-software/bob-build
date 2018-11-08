# Common properties for generate modules

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
The command that is to be run for this source generation.
Substitutions can be made in the command, by using
`$name_of_var`. A list of substitutions that can be used:

- `$gen_dir` - the path to the directory which belongs to this source generator
- `$in` - the path to the sources - space-delimited
- `$out` - the path to the targets - space-delimited
- `$args` - the value of "args" - space-delimited
- `$tool` - the path to the tool
- `$host_bin` - the path to the binary that is produced by the host_bin module
- `$(name)_dir` - the build directory for each dep in generated_dep
- `$src_dir` - the path to the project source directory - this will be different
  than the build source directory for Android.
- `$module_dir` - the path to the module directory

----
### **bob_generated.tool** (required)
A path to the tool that is to be used in `cmd`. If `$tool` is in
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
used in the command through `$(name_of_dependency)_dir` (that is, the variable's
name is the name of the dependency, with the `_dir` suffix).

----
### **bob_generated.module_srcs** (optional)
A list of other modules that this generator depends on.
The dependencies will be added to the list of srcs.

----
### **bob_generated.args** (optional)
A list of `args` that will be space separated and added to the `cmd`.

----
### **bob_generated.console** (optional)
This will use Ninja's [console pool](https://ninja-build.org/manual.html#_the_literal_console_literal_pool)
When `true` one job will run at a time - they won't be concurrent.

----
### **bob_generated.export_gen_include_dirs** (optional)
Additional include paths to add for modules that use `generated_headers`. This
will be defined relative to the module-specific build directory.

----
### **bob_generated.flag_defaults** (optional)
Generated sources may wish to access the build flags being used for "normal"
library or executable modules. `flag_defaults` should contain the name of a
`bob_defaults` module, whose flags will be accessible from this one, by
allowing extra variables to be used in `bob_generated.cmd`: `ar`, `cc`,
`cflags`, `conlyflags`, `cxx`, and `cxxflags`.

----
### **bob_generated.target** (required)
The target type - must be either `host` or `target`. This is to choose between
the host and target variant of the `bob_defaults` specified in
`bob_generate.flag_defaults`.
