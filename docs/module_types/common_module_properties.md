Common properties for modules
=============================

Executables, Shared Libraries and Static Libraries use some or all of the
properties described here.

Modules using each property will be referred to as `bob_module`.

----
### **bob_module.out** (optional)
Alternate output name, used for the file name and Android rules.

----
### **bob_module.enabled** (optional)
Used to disable the generation of build rules.
If this is set to false, no build rule will be generated.

**Default value:** true

----
### **bob_module.build_by_default** (optional)
Whether it is built by default in a build with no
targets requested.

**Default value:** true for `bob_shared_library`, `bob_binary`.
**Default value:** false for `bob_static_library`.

----
### **bob_module.name** (required)
The unique identifier that can be used to refer to this module.

All names must be be unique for the whole of the Android build system.

Shared library names must begin with `lib`.

----
### **bob_module.defaults** (optional)

The list of default properties that should prepended
to all configuration.

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

In this example, the `bob_binary` will
have the flags `-DGLOBAL_FLAG=1 -DMYFLAG=2 -DBINARY_FLAG=3`. The ordering is
from least to most-specific: Bob will traverse the graph of defaults which a
module (like `my_binary`) depends on, and prepend flags in breadth-first order.
This means that flags specified in the actual module can override options set
in defaults, because later flags usually take precedence.

----
### **bob_module.srcs** (optional)
The list of source files. Wildcards can be used, although they are suboptimal;
each directory in which a wildcard is used will have to be rescanned at every
build.

Source files, given with the parameter `srcs`, are
relative to the directory of the `build.bp` file.

----
### **bob_module.exclude_srcs** (optional)
The `exclude_srcs` property will remove files from `srcs`, for example things
which were picked up by a glob. `exclude_srcs` also supports wildcards, with
the same caveat as `srcs`.

----
### **bob_module.add_to_alias** (optional)
Adds this module to an alias. This is equivalent to adding `bob_module.name` to
the alias's `srcs` list.

----
### **bob_module.cflags** (optional)
Flags used for C/C++ compilation.

Flags can be added with the `cflags` parameter.
Note that defines are not specially treated, and must
thus be added as flags using `"-DCOLOR_DEF=blue"`

----
### **bob_module.export_cflags** (optional)
Flags exported to modules which depend on the current one. These will
only propagate one level. For example if we have three libraries with
dependencies `libA -> libB -> libC`, flags inside `libC`'s
`export_cflags` will only be exported to `libB`.

If `libB` wishes to propagate `libC`'s flags to `libA`, it should add
`libC` to its `reexport_libs` list.

----
### **bob_module.conlyflags** (optional)
Flags used for C compilation.

----
### **bob_module.cxxflags** (optional)
Flags used for C++ compilation.

----
### **bob_module.asflags** (optional)
Flags used for assembly compilation.

----
### **bob_module.ldflags** (optional)
Flags used for linking. Unlike `ldlibs`, `ldflags` is added to the _start_ of
the linker command-line.

---
### **bob_module.header_libs** (optional)
The list of header libraries whose include directories this library should import.

---
### **bob_module.export_header_libs** (optional)
On static and shared libraries, the list of header libraries whose include
directories this library should both import and export to its users.

----
### **bob_module.static_libs** (optional)
The list of static lib modules that this library depends on.

----
### **bob_module.shared_libs** (optional)
The list of shared lib modules that this library depends on.

----
### **bob_module.reexport_libs** (optional)
The exported cflags and includes of dependencies listed in `reexport_libs` are
also exported to users of the current module. The primary use case is where this
module's headers include the headers of its dependencies, leaking the
identifiers.

----
### **bob_module.ldlibs** (optional)
Linker flags required to link to the necessary system libraries. Unlike
`ldflags`, this is added to the _end_ of the linker command-line.

----
### **bob_module.generated_headers** (optional)
The list of modules that generate extra headers for this module.
We can use name of:
- `bob_generate_source`
- `bob_transform_source`

----
### **bob_module.generated_sources** (optional)
The list of modules that generate extra source files for this module.
We can use name of:
- `bob_generate_source`
- `bob_transform_source`

----
### **bob_module.generated_deps** (optional)
The list of modules that generate output required by the
build wrapper.
We can use name of:
- `bob_generate_source`
- `bob_transform_source`

----
### **bob_module.tags** (optional)
Values to use on Android for `LOCAL_MODULE_TAGS`, defining
which builds this module is built for.

----
### **bob_module.owner** (optional)
Value to use on Android for `LOCAL_MODULE_OWNER`

----
### **bob_module.include_dirs** (optional)
The list of include dirs to use that is relative
to the source directory.

----
### **bob_module.local_include_dirs** (optional)
The list of include dirs to use that is relative to the
build.bp file.

Include directories are added with the `local_include_dirs`
parameter. The paths added are relative to the directory
of the `build.bp` file. This can be used for generated headers
or if there are system headers not on the normal include path.

----
### **bob_module.export_local_include_dirs** (optional)
Include dirs (relative to module directory) to be
exported to modules linking with the current library.

----
### **bob_module.export_include_dirs** (optional)
Include dirs (path relative to root) to be exported
to modules linking with the current library.

----
### **bob_module.build_wrapper** (optional)
Wrapper for all build commands (object file
compilation **and** linking). This can be used, for example, to enable `ccache`:

```bp
bob_defaults {
    name: "toplevel_defaults",
    enable_ccache: {
        build_wrapper: "ccache",
    },
}
```

----
### **bob_module.build_wrapper_deps** (optional)
Files that the wrapper depends on.

----
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

----
### **bob_module.install_group** (optional)
Module specifying an installation directory.

----
### **bob_module.install_deps** (optional)
Other modules which must be installed alongside this, for example resources
specified in a `bob_resource`. Other libraries and binaries can also be
mentioned here, as well as any generated module.

----
### **bob_module.relative_install_path** (optional)
Path to install to, relative to the install_group's path.

----
### **bob_module.post_install_tool** (optional)
Script used during post install.

----
### **bob_module.post_install_cmd** (optional)
Command to execute on file(s) after they are installed. The following variables
are substituted into the command:

- `${tool}` - the tool specified in `bob_module.post_install_tool`.
- `${out}` - the output file(s) of the current module.
- `${bob_config}` - the Bob configuration file.

----
### **bob_module.target_supported** (optional)
If true, the module will be built using the target toolchain. `host_supported`
and `target_supported` can both be enabled. In this case, the module will be
built twice, once for each toolchain.

**Default value:** true

----
### **bob_module.host_supported** (optional)
If true, the module will be built using the host toolchain. `host_supported`
and `target_supported` can both be enabled. In this case, the module will be
built twice, once for each toolchain.

**Default value:** false

----
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
