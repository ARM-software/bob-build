Module: bob_generate_shared_library, bob_generate_static_library, bob_generate_binary
========================================================================================

This target generates a shared library, a static library, or a binary
using a custom command instead of via the default compiler and linker.
The libraries can be linked to other modules using the normal
properties that reference shared/static libraries. Headers for the
libraries can be generated at the same time.

The module type is `bob_generate_shared_library`,
`bob_generate_static_library`, or `bob_generate_binary`.

These module types support [features](../features.md)

## Full specification of `bob_generate_[shared|static]_library` and `bob_generate_binary` properties

For general common properties please
[check detailed documentation](common_module_properties.md).

For generate common properties please
[check detailed documentation](common_generate_module_properties.md).

```bp
bob_generate_static_library {
    // see bob_generate_shared_library
}
```

```bp
bob_generate_binary {
    // see bob_generate_shared_library
}
```

```bp
bob_generate_shared_library {
    name: "custom_name",
    srcs: ["src/a.cpp", "src/b.cpp", "src/common/*.cpp"],
    exclude_srcs: ["src/common/skip_this.cpp"],
    implicit_srcs: ["foo/*.tmpl],
    exclude_implicit_srcs: ["foo/a.tmpl"],
    headers: ["my.h"],

    enabled: false,
    build_by_default: true,

    add_to_alias: ["bob_alias.name"],

    cmd: "python ${tool} ${args} ${in}",
    tools: ["my_script.py"],

    host_bin: "name_of_host_binary",
    tags: ["optional"],

    generated_deps: ["bob_generate_source.name"],
    generated_sources: ["bob_generate_source.name"],

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
    rsp_content: "${in}",
}
```

----
### **bob_generate_*.implicit_srcs** (optional)

List of implicit sources. Implicit sources are input files that do not get
mentioned on the command line, and are not specified in the explicit sources.

----
### **bob_generate_*.exclude_implicit_srcs** (optional)

Used in combination with glob patterns in `implicit_srcs` to exclude
files that are not sources.

----
### **bob_generate_*.headers** (optional)

List of headers that are created (if any).
