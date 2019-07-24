Module: bob_generate_shared_library & bob_generate_static_library
=================================================================

This target generates a shared/static library and headers using a
custom command, instead of via the default compiler and linker. The
library can be linked to other modules using the normal properties
that reference shared/static libraries.

The module type is `bob_generate_shared_library` or `bob_generate_static_library`.

`bob_generate_static_library` supports [features](../features.md)

## Full specification of `bob_generate_shared_library` and `bob_generate_static_library`
For general common properties please
[check detailed documentation](common_module_properties.md).

For generate common properties please
[check detailed documentation](common_generate_module_properties.md).

```bp
bob_generate_static_library {
    // see below
}
```

```bp
bob_generate_shared_library {
    name: "custom_name",
    srcs: ["src/a.cpp", "src/b.cpp", "src/common/*.cpp"],
    exclude_srcs: ["src/common/skip_this.cpp"],
    headers: ["my.h"],

    enabled: false,
    build_by_default: true,

    add_to_alias: ["bob_alias.name"],

    cmd: "python ${tool} ${args} ${in}",
    tool: "my_script.py",

    host_bin: "name_of_host_binary",
    tags: ["optional"],

    module_deps: ["bob_generate_source.name"],
    module_srcs: ["bob_generate_source.name"],

    args: ["-i graphic/ui.h"],

    console: true,

    export_gen_include_dirs: ["."],

    flag_defaults: ["bob_default.name"],

    target: "host",

    install_group: "bob_install_group.name",
    install_deps: ["bob_resource.name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${args} ${out}",
    post_install_args: ["arg1", "arg2"],
}
```

----
### **bob_generate_shared_library.headers** or **bob_generate_static_library.headers** (optional)
List of headers that are created (if any).
