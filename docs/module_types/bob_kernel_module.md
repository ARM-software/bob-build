# Module: bob_kernel_module
This target is a kernel module (.ko) that will be built as an
external kernel module.

A kernel module is defined using the build rule
`bob_kernel_module`. This invokes an out-of-tree kernel build for
the module in question.

`bob_kernel_module` supports [features](../features.md)

To avoid cluttering the source directory, the `bob_kernel_module` will
copy all of its sources to the build directory before invoking Kbuild.
This means that *all* files in the module directory, including
`Kbuild`, `Makefile`, and `.h` files, must be included in the
`bob_kernel_module.srcs` list.

Furthermore, the `build.bp` containing the `bob_kernel_module`
definition must be in the same directory as the main `Kbuild` file for
that module.

## Full specification of `bob_kernel_module` properties
Most properties are optional.

For general common properties please [check detailed documentation](common_module_properties.md).

```bp
bob_kernel_module {
    name: "custom_name",
    srcs: ["Kbuild", "my_module.c", "*.h"],
    exclude_srcs: ["skip_this_header.h"],

    enabled: false,
    build_by_default: true,

    add_to_alias: ["bob_alias.name"],

    defaults: ["bob_default.name"],

    cflags: ["-DDEBUG=1", "-Wall"],

    tags: ["optional"],
    owner: "company_name",

    include_dirs: ["include/"],
    local_include_dirs: ["include/"],

    kbuild_options: ["CONFIG_MY_OPTION=y"],
    extra_symbols: ["bob_kernel_module.name"],
    make_args: ["SOME_MAKE_VARIABLE=3"],
    kernel_dir: "{{.kernel_dir}}",
    kernel_compiler: "{{.kernel_prefix}}",

    install_group: "bob_install_group.name",
    install_deps: ["bob_resource.name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${out} ARGS...",

    // features available
}
```

----
### **bob_kernel_module.kbuild_options** (optional)
Linux kernel config options to emulate. These are passed to Kbuild in
the `make` command-line, and set in the source code via
`EXTRA_CFLAGS`. These should usually include the `CONFIG_` prefix,
although it is possible to omit this if required.

----
### **bob_kernel_module.extra_symbols** (optional)
Kernel modules which this module depends on.

----
### **bob_kernel_module.make_args** (optional)
Arguments to pass to kernel make invocation.

----
### **bob_kernel_module.kernel_dir** (optional)
Kernel directory location.

----
### **bob_kernel_module.kernel_compiler** (optional)
Compiler prefix for kernel build.
