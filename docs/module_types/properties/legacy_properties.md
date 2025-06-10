# Common Legacy Properties

These properties can only be set on legacy modules.

## `defaults`

List of [`bob_defaults`](bob_defaults.md); default is `[]`

The configuration specified by a listed `bob_defaults` module is
allowed to conflict with another listed module, though this should
generally be avoided. A conflict can be for a single-valued property
like `enabled` or `build_wrapper`, or in a list property where a
particular element may have an opposite meaning (in `cflags`) or
modify search behaviour (in `include_dirs`).

Configuration in later `bob_defaults` will take priority over
configuration in earlier ones. In the following example, any conflicts
between the `generic` and `specific` modules will end up following
what is set by `specific`.

    defaults: [
        "generic",
        "specific",
    ],

Configuration in the current module will override configuration set
within a `bob_defaults`. A consequence of this is that where
`bob_defaults` are nested, the leaf modules have least priority and
get overriden.

### Examples:

```bp
bob_default {
    name: "global_default",
    cflags: ["-DGLOBAL_FLAG=1"],
}

bob_default {
    name: "my_default_1",
    defaults: ["global_default"],
    cflags: ["-DMY_FLAG=2"],
}

bob_binary {
    name: "my_binary",
    defaults: ["my_default_1"],
    cflags: ["-DBINARY_FLAG=3"],
}
```

In this example, `my_binary` will have the flags `-DGLOBAL_FLAG=1 -DMYFLAG=2 -DBINARY_FLAG=3`.

## `pgo`

Bob has rudimentary support for Profile-Guided Optimization when using the
Android.bp backend. Properties in the `pgo` block can be set, and will be
used as the values for the corresponding Soong properties, as described
[here](https://source.android.com/devices/tech/perf/pgo).

```bp
bob_binary {
    name: "pgo_optimized_binary",
    srcs: [...],
    pgo: {
        benchmarks: [
            "benchmark1",
            "benchmark2",
        ],
        profile_file: "pgo_optimized_binary.profdata",
        enable_profile_use: true,
        cflags: ["-DENABLED_WHEN_PGO_USED"],
    }
}
```

The `instrumentation` Soong property will be automatically set to `true` if
`profile_file` is set. Similarly, the only supported value of Soong's `sampling`
field is `false`, so it is not settable in Bob.

On backends other than Android.bp, these properties will be ignored.

## `strip`

When set, strip symbols and debug information from libraries and
binaries. This is a separate stage that occurs after linking and
before post install.

On Android, its infrastructure is used to do the stripping. If not
enabled, follow Android's default behaviour.

## `local_include_dirs`

List of strings; default is `[]`

A list of include directories to use. These are relative to the
`build.bp` containing the module definition, and expected to be within
the source hierarchy.

## `export_include_dirs`

List of strings; default is `[]`

A list of include directories, similar to `include_dirs`. These
directories also get added to the include paths of any module that
links to the current library.

Not supported on Android. Use `export_local_include_dirs` instead.

## `export_local_include_dirs`

List of strings; default is `[]`

A list of include directories to use, similar to
`local_include_dirs`. These directories also get added to the include
paths of any module that links to the current library.

## `export_system_include_dirs`

List of strings; default is `[]`

The same as `export_include_dirs` except downstream
dependencies use `-isystem` instead of `-I` for paths specified in this
attribute.

Not supported on Android. Use `export_local_system_include_dirs` instead.

## `export_local_system_include_dirs`

List of strings; default is `[]`

The same as `export_local_include_dirs` except downstream
dependencies use `-isystem` instead of `-I` for paths specified in this
attribute.

## `build_wrapper`

String; default is `none`.

Wrapper for all build commands (object file compilation **and**
linking). If the first word looks like a relative path (it doesn't
start with '/' but contains '/' characters), it is assumed that the
script is in the project directory.

This can be used, for example, to enable `ccache`:

```bp
bob_defaults {
    name: "toplevel_defaults",
    enable_ccache: {
        build_wrapper: "ccache",
    },
}
```

## `forwarding_shlib`

Boolean; default is `false`

This is a shared library that pulls in one or more shared libraries to
resolve symbols that the binary needs. This is useful where a named
library is the standard library to link against, but the
implementation may exist in another library.

Only valid on `bob_shared_library`.

Currently we need to link with `-Wl,--copy-dt-needed-entries`. This
makes the binary depend on the implementation library, and requires
the BFD linker.

This isn't guaranteed to work on Android.

## `add_lib_dirs_to_rpath`

Boolean; default is `false`

If true, the module's shared libraries' directories will be added to
its DT_RUNPATH entry. This allows the libraries to be found at runtime
without setting LD_LIBRARY_PATH or putting them in a standard system
location like `/usr/`."

## `install_group`

Target; default is `none`

Module name of a `bob_install_group` specifying an installation directory.

## `install_deps`

List of targets; default is `[]`

Other modules which must be installed alongside this, for example resources
specified in a `bob_resource`. Other libraries and binaries can also be
mentioned here, as well as any generated module.

## `debug_info`

Target; default is `none`

Module name of a `bob_install_group` specifying an installation
directory for debug information. If supplied, debug information will
be placed in a separate file (Linux only).

## `post_install_cmd`

String; default is `none`

Command to execute on file(s) after they are installed. The following variables
are substituted into the command:

- `${tool}` - the tool specified in `bob_module.post_install_tool`.
- `${out}` - the output file(s) of the current module.
- `${args}` - arguments from `post_install_args`

Not supported on Android.

## `post_install_args`

List of strings; default is `[]`

Arguments to insert into `post_install_cmd`. This allows arguments to
added based on features and defaults. Not supported on Android.

## `ldlibs`

List of strings; default is `[]`

Linker flags required to link to the necessary system libraries. Unlike
`ldflags`, this is added to the _end_ of the linker command-line.
These are propagated to the closest linking object when specified on static
libraries.

## `generated_headers`

List of targets; default is `[]`

The list of modules that generate extra headers for this module.
We can use name of:

- `bob_generate_source`
- `bob_transform_source`

## `generated_sources`

List of targets; default is `[]`

The list of modules that generate extra source files for this module.
We can use name of:

- `bob_generate_source`
- `bob_transform_source`

## `generated_deps`

List of targets; default is `[]`

The list of modules that generate output required by the
build wrapper.
We can use name of:

- `bob_generate_source`
- `bob_transform_source`

## `export_header_libs`

List of targets; default is `[]`

On static and shared libraries, the list of header libraries whose include
directories this library should both import and export to its users.

## `static_libs`

List of targets; default is `[]`

The list of static lib modules that this library depends on.
These are propagated to the closest linking object when specified on static
libraries.
`static_libs` is an indication that this module is using a static library, and
users of this module need to link against it.

## `whole_static_libs`

The `whole_static_libs` property allows a library to completely include the
contents of another. For example, if the above example was changed as follows:

```bp
bob_static_library {
    name: "libA",
    whole_static_libs: ["libB"],
    srcs: ["a.c"],
}
```

...then `libA.a` would contain _two_ object files - `a.o` and `b.o`. The link
command for `binary_using_libA` would then _only_ mention `libA`.

### Circular dependencies

The main reason for a 'parent' library to use `whole_static_libs` is circular
dependencies.

Suppose something inside `libB` now calls function `a()` in `libA`. The link
order needs to be such that:

- `libA` is before `libB`, because `libA` requires function `b()`, AND:
- `libB` is before `libA`, because `libB` requires function `a()`.

This is clearly impossible. The situation can be resolved by creating a new
static library, which can hold the contents of `libA` and `libB`
simultaneously. This will be passed to the linker instead of `libA` or `libB`,
enabling the link to succeed with the mutually-dependent components.

```bp
bob_static_library {
    name: "libAandB",
    whole_static_libs: ["libA", "libB"],
}
```

---

## `shared_libs`

List of targets; default is `[]`

The list of shared lib modules that this library depends on.
These are propagated to the closest linking object when specified on static
libraries.
`shared_libs` is an indication that this module is using a shared library, and
users of this module need to link against it.

## `reexport_libs`

List of targets; default is `[]`

The exported cflags and includes of dependencies listed in `reexport_libs` are
also exported to users of the current module. The primary use case is where this
module's headers include the headers of its dependencies, leaking the
identifiers.

## `header_libs`

List of targets; default is `[]`

The list of header libraries whose include directories this library should import.

## `exclude_srcs`

The `exclude_srcs` property will remove files from `srcs`, for example things
which were picked up by a glob. `exclude_srcs` also supports wildcards, with
the same caveat as `srcs`.

## `add_to_alias`

Adds this module to an alias. This is equivalent to adding `bob_module.name` to
the alias's `srcs` list.

## `cflags`

List of strings; default is `[]`

Flags used for C/C++ compilation.

Flags can be added with the `cflags` parameter.
Note that defines are not specially treated, and must
thus be added as flags using `"-DCOLOR_DEF=blue"`

Double quotes (") need to be escaped with backslash (\) to prevent the
blueprint parser consuming them. As with any string property, Go
templates can be used. Otherwise each flag should be written
as the C compiler expects to see it in its argument list i.e. without
shell escaping. Expansion of environment variables, ninja variables,
or make variables is not possible.

The [`add_if_supported`](../strings.md#add_if_supported) function can be
used to add a compiler argument only if it is supported.

For example to define a string literal:

```
    cflags: ["-DCOLOR_DEF=\"blue\""]
```

The [`match_srcs`](../strings.md#match_srcs) function can be used in
this property to reference files listed in `srcs`.

## `export_cflags`

List of strings; default is `[]`

C/C++ flags exported to modules which depend on the current one.

These will only propagate one level. For example if we have three libraries with
dependencies `libA -> libB -> libC`, flags inside `libC`'s
`export_cflags` will only be exported to `libB`.

If `libB` wishes to propagate `libC`'s flags to `libA`, it should add
`libC` to its `reexport_libs` list.

Also see `cflags`.
Note that we do not support [`match_srcs`](../strings.md#match_srcs)
function for `export_cflags`.

## `asflags`

List of strings; default is `[]`

Flags used for assembly compilation.

Double quotes (") need to be escaped with backslash (\) to prevent the
blueprint parser consuming them. As with any string property, Go
templates can be used. Otherwise each flag should be written as the
assembler expects to see it in its argument list i.e. without shell
escaping. Expansion of environment variables, ninja variables, or make
variables is not possible.

## `ldflags`

List of strings; default is `[]`

Flags used for linking. Unlike `ldlibs`, `ldflags` is added to the _start_ of
the linker command-line.

Double quotes (") need to be escaped with backslash (\) to prevent the
blueprint parser consuming them. As with any string property, Go
templates can be used. Otherwise each flag should be written
as the linker expects to see it in its argument list i.e. without
shell escaping. Expansion of environment variables, ninja variables,
or make variables is not possible.

The [`match_srcs`](../strings.md#match_srcs) function can be used in
this property to reference files listed in `srcs`.

## `export_ldflags`

List of strings; default is `[]`

Linker flags exported to modules which depend on the current one.

## `cmd`

String; required

The command that is to be run for this module. Bob supports various
substitutions in the command, by using `${name_of_var}`. The
available substitutions are:

- `${in}` - space-delimited list of source (input) paths
- `${out}` - space-delimited list of target (output) paths
- `${depfile}` - the path for the generated dependency file
- `${rspfile}` - the path to the RSP file, if `rsp_content` is set
- `${args}` - the value of `args` - space-delimited
- `${tool}` - the path to the first script specified in `tools`
- `${tool <label>}` - the path to the script with name `<label>` specified in `tools`
- `${host_bin}` - the path to the binary specified by `host_bin`
- `${module_dir}` - the path this module's source directory
- `${gen_dir}` - the path to the output directory for this module
- `${(name)_out}` - the outputs of the `generated_deps` dependency with `name`
- `${src_dir}` - the path to the project source directory - this will be different
  than the build source directory for Android.
- `${bob_config}` - the Bob configuration file. When used, a depfile must be
  generated naming the config file as a dependency to ensure the rule is
  correctly rerun when the configuration changes.
- `${bob_config_json}` - the Bob configuration JSON file, intended for use
  by tools that just need to read configuration values without having to know
  about the config system. When used, a depfile must be generated naming the
  config file as a dependency to ensure the rule is correctly rerun when the
  configuration changes.

The value in `cmd` is executed by the shell. Compound shell
expressions and expansions can be used, though we recommend keeping
commands simple. If double quotes (") need to be on the shell command
line, they should be escaped with backslash (\) to get through the
blueprint parser. Where a `$` needs to be evaluated by the shell (for
example to expand an environment variable) use `$$`.

The [`match_srcs`](../strings.md#match_srcs) function can be used in
this property to reference files listed in `srcs`.

## `tools`

List of strings. Default is `[]`

A path to the tools that are to be used in `cmd`. If `${tool}` is in
the command variable, then this will be replaced with the path to
the first tool on the list. To refer to the other tools provide its name
as `${tool example.sh}` for the `example.sh` specified in the list.

## `generated_deps`

A list of other modules that this generator depends on. The dependencies can be
used in the command through `${(name_of_dependency)_out}` (that is, the variable's
name is the name of the dependency, with the `_out` suffix).

## `generated_sources`

A list of other modules that this generator depends on.
The dependencies will be added to the list of srcs.

<!-- TODO: add docs for MTE props -->
