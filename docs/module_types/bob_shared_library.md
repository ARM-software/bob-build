Module: bob_shared_library
==========================

Used to create a shared library i.e. `.so` file.

## Full specification of `bob_shared_library` properties
`bob_shared_library` supports [features](../features.md)

Most properties are optional. For detailed documentation
please go to [common module properties](common_module_properties.md).

```bp
bob_shared_library {
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

    static_libs: ["bob_static_lib.name"],

    shared_libs: ["bob_shared_lib.name"],

    reexport_libs: ["bob_shared_lib.name", "bob_static_lib.name"],
    whole_static_libs: ["bob_static_lib.name"],

    ldlibs: ["-lz"],

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

    forwarding_shlib: true,
    add_lib_dirs_to_rpath: true,

    install_group: "bob_install_group.name",
    install_deps: ["bob_resource.name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${args} ${out}",
    post_install_args: ["arg1", "arg2"],
}
```

----
### **bob_shared_library.whole_static_libs** (optional)

Static libraries linked with a shared library using `whole_static_libs` will be
linked with the `-Wl,--whole-archive` linker flag. This ensures that the entire
contents of the static libraries are include in the shared library. Without
this, the linker may remove unused object files.

This will include all the static libs' objects in the shared library (as
opposed to normal static linking, which will only include unresolved symbols).
