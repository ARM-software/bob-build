Module: bob_filegroup
====================

This target lists a collection of source files that can be re-used in other targets. It exists
to enforce no relative uplinks and to closer align to Bazel.

## Full specification of `bob_filegroup` properties

For general common properties please
[check detailed documentation](common_module_properties.md).

```bp
bob_filegroup {
    name: "custom_name",

    srcs: ["src/a.cpp", "src/b.cpp", "src/common/*.cpp"],
    filegroup_srcs: ["other_filegroup_module"],
}
```

----
### **bob_filegroup.name** (required)

The unique identifier that can be used to refer to this module.

----
### **bob_filegroup.srcs** (optional)

Source files to add to other modules that depend upon this.

----
### **bob_filegroup.filegroup_srcs** (optional)

Other filegroups this filegroup depends on. Their sources are also
appended to the sources of any module that depends upon this module.
