Module: bob_generate_source
===========================

This target generates source code (headers or C files). A single
module can generate multiple outputs from common inputs, by running a
custom command.

The command will be run once - with `$in` being the paths in
`srcs` and `$out` being the paths in `out`.
The source and tool paths should be relative to the directory of the
`build.bp` containing the `bob_generate_source`.

## Full specification of `bob_generate_source` properties
For general common properties please
[check detailed documentation](common_module_properties.md).

For generate common properties please
[check detailed documentation](common_generate_module_properties.md).

```bp
bob_generate_source {
    name: "custom_name",
    srcs: ["src/a.cpp", "src/b.cpp", "src/common/*.cpp"],
    exclude_srcs: ["src/common/skip_this.cpp"],

    out: ["my_out.cpp"],
    depfile: true,
    implicit_srcs: ["foo/scatter.scat"],

    enabled: false,
    build_by_default: true,

    add_to_alias: ["bob_alias.name"],

    cmd: "python ${tool} ${args} ${in} -d ${depfile}",
    tool: "my_script.py",

    host_bin: "clang-tblgen",
    tags: ["optional"],

    module_deps: ["bob_generate_source.name"],
    module_srcs: ["bob_generate_source.name"],

    args: ["-i graphic/ui.h"],

    console: true,

    export_gen_include_dirs: ["."],
    encapsulates: ["bob_generate_source.name"],

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
### **bob_generate_source.out** (required)
The list of files that will be output.

----
### **bob_generate_source.implicit_srcs** (optional)
List of implicit sources. Implicit sources are input files that do not get
mentioned on the command line, and are not specified in the explicit sources.

----
### **bob_generate_source.implicit_outs** (optional)
List of implicit outputs. Implicit outputs are output files that do not get
mentioned on the command line.
