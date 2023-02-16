# Common properties for modules

Executables, Shared Libraries and Static Libraries use some or all of the
properties described here.

Modules using each property will be referred to as `bob_module`.

---

### **bob_module.out** (optional)

Alternate output name, used for the file name and Android rules.

---

### **bob_module.enabled** (optional)

Used to disable the generation of build rules.
If this is set to false, no build rule will be generated.

**Default value:** true

---

### **bob_module.build_by_default** (optional)

Whether it is built by default in a build with no
targets requested.

**Default value:** true for `bob_shared_library`, `bob_binary`.
**Default value:** false for `bob_static_library`.

---

### **bob_module.name** (required)

The unique identifier that can be used to refer to this module.

All names must be be unique for the whole of the Android build system.

Shared library names must begin with `lib`.

---

### **bob_module.defaults** (optional)

A list of [`bob_defaults`](bob_defaults.md) modules containing
configuration required by this module.

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

#### Examples:

```bp
bob_default {
    name: "global_default",
    cflags: ["-DGLOBAL_FLAG=1"],
    // ...
}

bob_default {
    name: "my_default_1",
    defaults: ["global_default"],
    cflags: ["-DMY_FLAG=2"],
    // ...
}

bob_binary {
    name: "my_binary",
    defaults: ["my_default_1"],
    cflags: ["-DBINARY_FLAG=3"],
    // ...
}
```

In this example, `my_binary` will have the flags `-DGLOBAL_FLAG=1 -DMYFLAG=2 -DBINARY_FLAG=3`.

---

### **bob_module.srcs** (optional)

The list of source files. Wildcards can be used, although they are suboptimal;
each directory in which a wildcard is used will have to be rescanned at every
build.

Source files, given with the parameter `srcs`, are relative to the
directory of the `build.bp` file.

An appropriate compiler will be invoked for each source file based on
its file extension. Files with an unknown extension are only allowed
if referenced by [`match_srcs`](../strings.md#match_srcs) usage within
the module, otherwise an error will be raised.

---

### **bob_module.exclude_srcs** (optional)

The `exclude_srcs` property will remove files from `srcs`, for example things
which were picked up by a glob. `exclude_srcs` also supports wildcards, with
the same caveat as `srcs`.

---

### **bob_module.filegroup_srcs** (optional)

The `filegroup_srcs` property will append files to `srcs` that are listed inside of
a `bob_filegroup` module. These are used so you can re-use collections of files &
to closer align to Bazel/Android.

---

### **bob_module.add_to_alias** (optional)

Adds this module to an alias. This is equivalent to adding `bob_module.name` to
the alias's `srcs` list.

---

### **bob_module.cflags** (optional)

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

---

### **bob_module.export_cflags** (optional)

Flags exported to modules which depend on the current one. These will
only propagate one level. For example if we have three libraries with
dependencies `libA -> libB -> libC`, flags inside `libC`'s
`export_cflags` will only be exported to `libB`.

If `libB` wishes to propagate `libC`'s flags to `libA`, it should add
`libC` to its `reexport_libs` list.

Also see `cflags`.
Note that we do not support [`match_srcs`](../strings.md#match_srcs)
function for `export_cflags`.

---

### **bob_module.conlyflags** (optional)

Flags used for C compilation. See `cflags`.

---

### **bob_module.cxxflags** (optional)

Flags used for C++ compilation. See `cflags`.

---

### **bob_module.asflags** (optional)

Flags used for assembly compilation.

Double quotes (") need to be escaped with backslash (\) to prevent the
blueprint parser consuming them. As with any string property, Go
templates can be used. Otherwise each flag should be written as the
assembler expects to see it in its argument list i.e. without shell
escaping. Expansion of environment variables, ninja variables, or make
variables is not possible.

---

### **bob_module.ldflags** (optional)

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

---

### **bob_module.header_libs** (optional)

The list of header libraries whose include directories this library should import.

---

### **bob_module.export_header_libs** (optional)

On static and shared libraries, the list of header libraries whose include
directories this library should both import and export to its users.

---

### **bob_module.static_libs** (optional)

The list of static lib modules that this library depends on.
These are propagated to the closest linking object when specified on static
libraries.
`static_libs` is an indication that this module is using a static library, and
users of this module need to link against it.

---

### **bob_module.shared_libs** (optional)

The list of shared lib modules that this library depends on.
These are propagated to the closest linking object when specified on static
libraries.
`shared_libs` is an indication that this module is using a shared library, and
users of this module need to link against it.

---

### **bob_module.reexport_libs** (optional)

The exported cflags and includes of dependencies listed in `reexport_libs` are
also exported to users of the current module. The primary use case is where this
module's headers include the headers of its dependencies, leaking the
identifiers.

---

### **bob_module.ldlibs** (optional)

Linker flags required to link to the necessary system libraries. Unlike
`ldflags`, this is added to the _end_ of the linker command-line.
These are propagated to the closest linking object when specified on static
libraries.

---

### **bob_module.generated_headers** (optional)

The list of modules that generate extra headers for this module.
We can use name of:

- `bob_generate_source`
- `bob_transform_source`

---

### **bob_module.generated_sources** (optional)

The list of modules that generate extra source files for this module.
We can use name of:

- `bob_generate_source`
- `bob_transform_source`

---

### **bob_module.generated_deps** (optional)

The list of modules that generate output required by the
build wrapper.
We can use name of:

- `bob_generate_source`
- `bob_transform_source`

---

### **bob_module.tags** (optional)

Values to use on Android for `LOCAL_MODULE_TAGS`, defining
which builds this module is built for.

---

### **bob_module.owner** (optional)

Value to use on Android for `LOCAL_MODULE_OWNER`
If set, then the module is considered proprietary. For the Soong plugin this will
usually be installed in the vendor partition.

---

### **bob_module.strip** (optional)

When set, strip symbols and debug information from libraries and
binaries. This is a separate stage that occurs after linking and
before post install.

On Android, its infrastructure is used to do the stripping. If not
enabled, follow Android's default behaviour.

---

### **bob_module.include_dirs** (optional)

A list of include directories to use. These are expected to be system
headers, and will usually be an absolute path. On Android these can be
relative to `$ANDROID_TOP`.

---

### **bob_module.local_include_dirs** (optional)

A list of include directories to use. These are relative to the
`build.bp` containing the module definition, and expected to be within
the source hierarchy.

---

### **bob_module.export_include_dirs** (optional)

A list of include directories, similar to `include_dirs`. These
directories also get added to the include paths of any module that
links to the current library.

Not supported on Android. Use `export_local_include_dirs` instead.

---

### **bob_module.export_local_include_dirs** (optional)

A list of include directories to use, similar to
`local_include_dirs`. These directories also get added to the include
paths of any module that links to the current library.

---

### **bob_module.export_system_include_dirs** (optional)

The same as `export_include_dirs` except downstream
dependencies use `-isystem` instead of `-I` for paths specified in this
attribute.

Not supported on Android. Use `export_local_system_include_dirs` instead.

---

### **bob_module.export_local_system_include_dirs** (optional)

The same as `export_local_include_dirs` except downstream
dependencies use `-isystem` instead of `-I` for paths specified in this
attribute.

---

### **bob_module.build_wrapper** (optional)

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

---

### **bob_module.forwarding_shlib** (optional)

This is a shared library that pulls in one or more shared libraries to
resolve symbols that the binary needs. This is useful where a named
library is the standard library to link against, but the
implementation may exist in another library.

Only valid on `bob_shared_library`.

Currently we need to link with `-Wl,--copy-dt-needed-entries`. This
makes the binary depend on the implementation library, and requires
the BFD linker.

This isn't guaranteed to work on Android.

---

### **bob_module.add_lib_dirs_to_rpath** (optional)

If true, the module's shared libraries' directories will be added to
its DT_RUNPATH entry. This allows the libraries to be found at runtime
without setting LD_LIBRARY_PATH or putting them in a standard system
location like `/usr/`."

**Default value:** false

---

### **bob_module.install_group** (optional)

Module name of a `bob_install_group` specifying an installation directory.

---

### **bob_module.install_deps** (optional)

Other modules which must be installed alongside this, for example resources
specified in a `bob_resource`. Other libraries and binaries can also be
mentioned here, as well as any generated module.

---

### **bob_module.relative_install_path** (optional)

Path to install to, relative to the install_group's path.

---

### **bob_module.debug_info** (optional)

Module name of a `bob_install_group` specifying an installation
directory for debug information. If supplied, debug information will
be placed in a separate file (Linux only).

---

### **bob_module.post_install_tool** (optional)

Script used during post install. Not supported on Android.bp.

---

### **bob_module.post_install_cmd** (optional)

Command to execute on file(s) after they are installed. The following variables
are substituted into the command:

- `${tool}` - the tool specified in `bob_module.post_install_tool`.
- `${out}` - the output file(s) of the current module.
- `${args}` - arguments from `post_install_args`

Not supported on Android.bp.

### **bob_module.post_install_args** (optional)

Arguments to insert into `post_install_cmd`. This allows arguments to
added based on features and defaults. Not supported on Android.bp.

---

### **bob_module.version_script** (optional)

Linker script used for [symbol versioning](../user_guide/libraries_2.md#markdown-header-symbol-versioning).
Only supported on binaries and shared libraries.

---

### **bob_module.target_supported** (optional)

If true, the module will be built using the target toolchain. `host_supported`
and `target_supported` can both be enabled. In this case, the module will be
built twice, once for each toolchain.

**Default value:** true

---

### **bob_module.host_supported** (optional)

If true, the module will be built using the host toolchain. `host_supported`
and `target_supported` can both be enabled. In this case, the module will be
built twice, once for each toolchain.

**Default value:** false

---

### **bob_module.target and bob_module.host** (optional)

Every property a module supports, except `name` and `defaults`, can also be
specified inside the `host: {}` or `target: {}` sections of a module
description. These properties will only be applied to the host or target
version of a module.

[Features](../features.md) can also be used inside the `host|target` sections.

```bp
bob_binary {
    name: "hello",
    host: {
        cflags: ["-DPLATFORM_NAME=host"],
    },
    target: {
        cflags: ["-DPLATFORM_NAME=target"],
        target_toolchain_clang: {
            cflags: ["-mtune=..."],
        },
    },
}
```

---

### **bob_module.pgo** (optional)

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
```

The `instrumentation` Soong property will be automatically set to `true` if
`profile_file` is set. Similarly, the only supported value of Soong's `sampling`
field is `false`, so it is not settable in Bob.

On backends other than Android.bp, these properties will be ignored.
