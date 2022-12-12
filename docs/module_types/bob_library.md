# Module: bob_library

!Warning, this target is experimental & the attributes/interface are likely to keep changing.

This target replaces `bob_static_library` & `bob_shared_library` to mimic Bazel and Soong in having a single library rule and being context aware to understand if a library should be statically built or dynamically.

## Full specification of `bob_library` properties

NOTE: Not all common properties are available to bob_library.

```bp
bob_library {
    name: "lib_forward_defines",
    srcs: ["src/libs/lib.cpp"],
    hdrs: ["src/libs/lib.h"],
    local_defines: ["LOCAL_DEFINE"],
    defines: ["FORWARDED_DEFINE"],
}
```

---

### **bob_library.name** (required)

The unique identifier that can be used to refer to this module.

---

### **bob_library.srcs** (optional)

Sources files to be compiled into the library that is built

---

### **bob_library.hdrs** (optional)

Headers that are a part of the library.

---

### **bob_library.local_defines** (optional)

Defines that are local to the module and are not added to modules that depend upon this.

---

### **bob_library.defines** (optional)

Defines that are included in the local module, and all modules that depend upon it. (Including transitively.)

---

### **bob_library.copts** (optional)

This options are included as cflags in the compile/link commands.

---

### **bob_library.deps** (optional)

A list of modules that this library depends on.

!Currently assumes all modules listed are static library dependencies, which is incorrect but used experimentally for testing.
