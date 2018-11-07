# Module: bob_static_lib
Used to create a static library, i.e. `.a` file.

## Static library dependencies

Static libraries may depend on functionality in other static libraries
(e.g. if `libA` calls function `b()` in `libB`, `libA` *depends on* `libB`).
Dependencies add restrictions on order the static libraries must appear in the
linker command-line - in this example, `libA` must appear *before* `libB`.

With shared libraries, this is handled automatically by the linker, because
dependencies can be encoded in the shared library file itself. However,
static libraries are simply collections of `.o` files, so this is not possible.
Instead, Bob provides two tools to automatically generate the correct library
order - `export_static_libs` and `whole_static_libs`.

## Full specification of `bob_static_library` properties
`bob_static_library` supports [features](../features.md)

Most properties are optional. For detailed documentation
please go to [common module properties](common_module_properties.md).

```bp
bob_static_library {
    name: "custom_name",
    srcs: ["src/a.cpp", "src/b.cpp", "src/common/*.cpp"],
    exclude_srcs: ["src/common/skip_this.cpp"],

    enabled: false,
    build_by_default: true,

    add_to_alias: ["bob_alias.name"],

    defaults: ["bob_default.name"],

    target_supported: true,
    target: { ... },

    host_supported: true,
    host: { ... },

    out: "alternate_output_name",

    cflags: ["-DDEBUG=1", "-Wall"],
    export_cflags: ["..."],

    cxxflags: ["..."],
    asflags: ["..."],
    conlyflags: ["..."],

    ldflags: ["..."],
    export_ldflags: ["..."],

    export_static_libs: ["libFooStatic"],

    export_shared_libs: ["..."],

    reexport_libs: ["bob_shared_lib.name", "bob_static_lib.name"],
    whole_static_libs: ["bob_static_lib.name"],

    export_ldlibs: ["-llog"],

    generated_headers: ["bob_generate_source.name"],
    generated_sources: ["bob_transform_source.name"],
    generated_deps: ["bob_generate_source.name"],

    tags: ["optional"],
    owner: "{{.android_module_owner}}",

    include_dirs: ["include/"],
    local_include_dirs: ["include/"],
    export_local_include_dirs: ["include/"],
    export_include_dirs: ["include/"],

    build_wrapper: "ccache",
    build_wrapper_deps: ["config.py"],

    install_group: "bob_install_group.name",
    install_deps: ["bob_resource.name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${out} ARGS...",
}
```

----
### **bob_module.export_ldflags** (optional)
Linker flags to be propagated to the top-level shared library or binary.

----
### **bob_static_lib.export_static_libs** (optional)
Static libraries can use the `export_static_libs` property to tell Bob about
any other static libraries they depend on. Bob will ensure that all static
libraries are placed earlier in the link order than their dependents. The
earlier example could therefore be resolved as follows:

```bp
bob_static_library {
    name: "libB",
    srcs: ["b.c"],
}

bob_static_library {
    name: "libA",
    export_static_libs: ["libB"],
    srcs: ["a.c"],
}

bob_binary {
    name: "binary_using_libA",
    static_libs: ["libA"],
}
```

The link command for `binary_using_libA` would contain `libA` first, then
`libB`.

----
### **bob_static_lib.whole_static_libs** (optional)

The `whole_static_libs` property allows a library to completely include the
contents of another. For example, if the above example was changed as follows:

```bp
bob_static_library {
    name: "libA",
    whole_static_libs: ["libB"],
    srcs: ["a.c"],
}
```

...then `libA.a` would contain *two* object files - `a.o` and `b.o`. The link
command for `binary_using_libA` would then *only* mention `libA`.

#### Circular dependencies
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

----
### **bob_static_lib.export_shared_libs** (optional)
The libraries mentioned here will be appended to `shared_libs` of the top-level
build object (shared library or binary) linking with this module.
`export_shared_libs` is an indication that this module is using a shared
library, and users of this module need to link it.


----
### **bob_static_lib.export_ldlibs** (optional)
Library dependency-related linker flags which should be added to the link
command of the top-level build object (shared library or binary).
