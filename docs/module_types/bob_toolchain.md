# Module: bob_toolchain

This module is never instantiated but provides toolchain flags
only to strict modules i.e. `bob_executable` & `bob_library`.

The toolchain module will export flags via flag provider and a
dependency tag of `ToolchainTag`.

## Full specification of `bob_toolchain` properties

`bob_toolchain` supports [features](../features.md)

```bp
bob_toolchain {
    name: "main_toolchain",
    cflags: [
        "-Wall",
        "-Werror",
    ],
    conlyflags: [
        "-std=c99",
    ],
    cppflags: [
        "-std=c++11",
    ],
    asflags: [
        "-Dasflag",
    ],
    ldflags: [
        "-Wl,--stats",
    ],
    target: {
        conlyflags: [
            "-Dconlyflag_target",
        ],
        ldflags: [
            "-Wl,--no-allow-shlib-undefined",
        ],
    },
    host: {
        cppflags: [
            "-Dcppflag_host",
        ],
        ldflags: [
            "-Wl,--allow-shlib-undefined",
        ],
    },
    always_enabled_feature: {
        cflags: [
            "-pedantic",
        ],
    },
}
```

---

### **bob_toolchain.cflags** (optional)

Flags that will be used for C and C++ compiles.

---

### **bob_toolchain.conlyflags** (optional)

Flags that will be used for C compiles

---

### **bob_toolchain.cppflags** (optional)

Flags that will be used for C++ compiles.

---

### **bob_toolchain.ldlags** (optional)

Flags that will be used for .S compiles.

---

### **bob_toolchain.aslags** (optional)

Flags that will be used for all link steps.

# Usage

To specify correct `bob_toolchain` dependency use `toolchain` property e.g.:

```bp
bob_toolchain {
    name: "main",
    ...
}

bob_library {
    name: "foo",
    ...
    toolchain: "main",
    ...
}

bob_executable {
    name: "app",
    ...
    toolchain: "main",
    ...
}
```