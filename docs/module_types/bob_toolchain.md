# Module: bob_toolchain

> ⚠ Warning, this target is experimental & the attributes/interface are likely to keep changing. ⚠

```bp
bob_toolchain {
    name, cflags, conlyflags, cppflags, asflags, ldflags, target, host, mte, tags
}
```

This module is never instantiated but provides toolchain flags
only to strict modules i.e. `bob_executable` & `bob_library`.

The toolchain module will export flags via flag provider and a
dependency tag of `ToolchainTag`.

Supports:

- [features](../features.md)

## Properties

|                                                      |                                                                                                                                                                                                                                                                                                                                                                                                       |
| ---------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name)       | String; required                                                                                                                                                                                                                                                                                                                                                                                      |
| [`target`](properties/common_properties.md#target)   | Property map; default is `{}`.                                                                                                                                                                                                                                                                                                                                                                        |
| [`host`](<(properties/common_properties.md#host)>)   | Property map; default is `{}`.                                                                                                                                                                                                                                                                                                                                                                        |
| [`cflags`](properties/legacy_properties.md#cflags)   | List of strings; default is `[]`<br>Flags used for C/C++ compilation.                                                                                                                                                                                                                                                                                                                                 |
| `conlyflags`                                         | List of strings; default is `[]`<br>Flags used for C compilation.<br>See [`cflags`](properties/legacy_properties.md#cflags)                                                                                                                                                                                                                                                                           |
| `cppflags`                                           | List of strings; default is `[]`<br>Flags used for C++ compilation.<br>See [`cflags`](properties/legacy_properties.md#cflags)                                                                                                                                                                                                                                                                         |
| [`asflags`](properties/legacy_properties.md#asflags) | List of strings; default is `[]`<br>Flags used for assembly compilation.                                                                                                                                                                                                                                                                                                                              |
| [`ldflags`](properties/legacy_properties.md#ldflags) | List of strings; default is `[]`<br>Flags used for linking.                                                                                                                                                                                                                                                                                                                                           |
| `mte`                                                | Property map; default is `{}`.<br>Flags to be used to enable the Arm Memory Tagging Extension.<br>Only supported on Android.<br>- **memtag_heap** - Memory-tagging, only available on arm64 if `diag_memtag_heap` unset or false, enables async memory tagging.<br>- **diag_memtag_heap** - Memory-tagging, only available on arm64 requires `memtag_heap`: true if set, enables sync memory tagging. |
| [`tags`](properties/common_properties.md#tags)       | List of strings; default is `[]`<br>This list of tags will be appended to any module using this toolchain configuration.                                                                                                                                                                                                                                                                              |

## Example

To specify correct `bob_toolchain` dependency use `toolchain` property e.g.:

```bp
bob_toolchain {
    name: "main",
}

bob_library {
    name: "foo",
    toolchain: "main",
}

```

## Default Behaviour

A `bob_toolchain` will be applied to the current directory scope and recursively into child directories, unless
overwritten by another `bob_toolchain`.

This toolchain is only applied to modules which support it.

Take the following project structure as an example:

```
.
├── build.bp >>> `bob_toolchain("A")`
├── inherits
│   ├── build.bp
│   └── overwrite
│       ├── build.bp >>> `bob_toolchain("B")`
│       └── parent
│           └── build.bp
```

In this case the mapping will be:

```
`.` -> `A`
`./inherits` -> `A`
`./inherits/overwrite` -> `B`
`./inherits/overwrite/parent` -> `B`
```

> `bob_toolchain` module attributes are overwritten, and **_not_** merged.
