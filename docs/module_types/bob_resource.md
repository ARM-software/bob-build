Module: bob_resource
====================

This target identifies files in the source tree which should be copied to
the installation directory, e.g. files which the project may
need while executing.

This will reference an `bob_install_group` so it gets copied to an appropriate location
relative to the binaries.

For the Soong plugin, the `install_path` set in the `bob_install_group` must be
prefixed by a known string to select an appropriate Android directory.
Currently `data/` and `etc/` are supported. The `owner` property also influences
where the files will be installed.

`bob_resource` supports [features](../features.md)

## Full specification of `bob_resource` properties
For general common properties please
[check detailed documentation](common_module_properties.md).

```bp
bob_resource {
    name: "custom_name",

    srcs: ["src/a.cpp", "src/b.cpp", "src/common/*.cpp"],
    exclude_srcs: ["src/common/skip_this.cpp"],

    enabled: false,
    build_by_default: true,

    add_to_alias: ["bob_alias.name"],

    install_group: "bob_install_group.name",
    install_deps: ["bob_resource.name"],
    relative_install_path: "unit/objects",
    post_install_tool: "post_install.py",
    post_install_cmd: "${tool} ${args} ${out}",
    post_install_args: ["arg1", "arg2"],

    tags: ["optional"],
    owner: "company_name",

    // features available
}
```

----
### **bob_resource.name** (required)
The unique identifier that can be used to refer to this module.

----
### **bob_resource.srcs** (optional)
Source files to copy to the installation directory.

----
### **bob_resource.add_to_alias** (optional)
Adds this module to an alias.

----
### **bob_module.owner** (optional)
Value to use on Android for `LOCAL_MODULE_OWNER`
If set, then the module is considered proprietary. For the Soong plugin this will
usually be installed in the vendor partition.