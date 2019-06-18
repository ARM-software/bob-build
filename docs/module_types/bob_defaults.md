Module: bob_defaults
====================

Defaults are used to share common settings between multiple modules.

`bob_defaults` can be used by `bob_static_lib`, `bob_shared_lib`, `bob_binary`,
and `bob_kernel_module`.

## Full specification of `bob_defaults` properties
Most properties are optional.

`bob_defaults` supports [features](../features.md)

For detailed documentation please go to [common module properties](docs/bp/module/common_module_properties.md).
For kernel module related stuff please check [bob_kernel_module](bob_kernel_module.md)

```bp
bob_defaults {
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

    static_libs: ["bob_static_lib.name"],
    export_static_libs: ["..."],

    shared_libs: ["bob_shared_lib.name"],
    export_shared_libs: ["..."],

    reexport_libs: ["bob_shared_lib.name", "bob_static_lib.name"],

    whole_static_libs: ["bob_shared_lib.name", "bob_static_lib.name"],

    ldlibs: ["-lz"],
    export_ldlibs: ["-llog"],

    generated_headers: ["bob_generate_source.name"],
    generated_sources: ["bob_transform_source.name"],
    generated_deps: ["bob_generate_source.name"],

    tags: ["optional"],
    owner: "my_company",

    include_dirs: ["include/"],
    local_include_dirs: ["include/"],
    export_local_include_dirs: ["include/"],
    export_include_dirs: ["include/"],

    build_wrapper: "ccache",
    forwarding_shlib: true,

    // kernel module related stuff
    kbuild_options: ["CONFIG_MY_OPTION=y"],
    extra_symbols: ["bob_kernel_module.name"],
    make_args: ["--ignore-errors"],
    kernel_dir: "/kernel/linux/",
    kernel_cross_compile: "prefix",
    kernel_cc: "target",
    kernel_hostcc: "host",
    kernel_clang_triple: "triple",
    // ^^ kernel module building related stuff

    install_group: "bob_install_group.name",
    install_deps: ["bob_resource.name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${out} ARGS...",

    // features available
}
```

----
# Examples

Here, the `-lncurses` flag is used with linking the `less` binary,
because it has been propagated through the default.

```bp
bob_defaults {
    name: "common_libs",
    ldlibs: ["-lncurses"],
}

bob_binary {
    name: "less",
    defaults: ["common_libs"],
    srcs: ["src/less.c"],
}
```

Defaults can be hierarchical by including other defaults. Here, the
`my_unit_test` binary will inherit both the `-Wall` and `-UNDEBUG` flags.

```bp
bob_defaults {
    name: "project_wide",
    cflags: ["-Wall"],
}

bob_defaults {
    name: "unit_test_defaults",
    defaults: ["project_wide"],
    cflags: ["-UNDEBUG"],
}

bob_binary {
    name: "my_unit_test",
    defaults: ["unit_test_defaults"],
    srcs: ["unit_test.c"],
}
```
