# Module: bob_import_cc_library

> Warning, this target is experimental and the attributes/interface are likely to keep changing.

```bp
bob_import_cc_library {
    name, src, includes, defines, target, linkopts
}
```

This target includes externally built C/C++ libraries in the Linux build graph. It is intended to support incremental migration to Bazel by allowing Bob modules to consume libraries that were built outside Bob.

## Properties

|                                                |                                                                                                                                                 |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                                                                                                                                |
| `src`                                          | Optional, path of the built library. Leave empty for a header-only library.                                                                     |
| `includes`                                     | List of include dirs needed to compile against the external library.                                                                            |
| `target`                                       | One of either `host` or `target`. Whether the library is built for host or target.                                                              |
| `defines`                                      | List of strings; default is `[]`<br>Defines that are included in the local module, and all modules that depend upon it. Including transitively. |
| [`linkopts`](properties/linkopts.md)           | List of strings; default is `[]`<br>List of additional flags to the linker command.                                                             |
