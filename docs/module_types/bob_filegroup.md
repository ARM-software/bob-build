# Module: bob_filegroup

This target lists a collection of source files that can be re-used in other targets. It exists
to enforce no relative uplinks and to closer align to Bazel.

## Full specification of `bob_filegroup` properties

For general common properties please
[check detailed documentation](common_module_properties.md).

```bp
// Simple filegroup with source file list.
bob_filegroup {
    name: "other_filegroup_module",
    srcs: ["src/a.cpp"],
}

bob_glob {
    name: "glob_group",
    srcs: ["src/common/*.cpp"],
}

bob_filegroup {
    name: "custom_name",
    srcs: [
        "src/b.cpp",
        ":other_filegroup_module",
        ":glob_group"
    ],
}
```

---

### **bob_filegroup.name** (required)

The unique identifier that can be used to refer to this module.

---

### **bob_filegroup.srcs** (optional)

A list of source files or other targets this filegroup includes.

---
