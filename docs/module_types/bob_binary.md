Module: bob_binary
==================

Target is an executable.

## Full specification of `bob_binary` properties
Most properties are optional.

`bob_binary` supports [features](../features.md)

For general common properties please [check detailed documentation](common_module_properties.md).

```bp
bob_binary {
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
    cxxflags: ["..."],
    asflags: ["..."],
    conlyflags: ["..."],

    ldflags: ["..."],
    ldlibs: ["-lz"],

    static_libs: ["bob_static_lib.name", "bob_generated_static.name"],
    shared_libs: ["bob_shared_lib.name", "bob_generated_shared.name"],

    generated_headers: ["module_name"],
    generated_sources: ["module_name"],
    generated_deps: ["module_name"],

    tags: ["optional"],
    owner: "company_name",

    include_dirs: ["include/"],
    local_include_dirs: ["include/"],

    build_wrapper: "ccache",

    add_lib_dirs_to_rpath: true,

    install_group: "bob_install_group.name",
    install_deps: ["module_name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${args} ${out}",
    post_install_args: ["arg1", "arg2"],

    // features available
}
```
