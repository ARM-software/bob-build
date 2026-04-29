# Module: bob_import_cc

> ⚠ Warning, this target is experimental & the attributes/interface are likely to keep changing. ⚠

```bp
bob_import_cc {
    name, src, includes, defines, target, linkopts
}
```

This target exists to include externally built libraries into the linux build graph. This is to better enable a future transition to bazel.

## Properties

|                                                |                                                                                                                                                   |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | --- |
| [`name`](properties/common_properties.md#name) | String; required                                                                                                                                  |
| `src`                                          | Optional, path of the built library, leave empty for a header only library.                                                                       |
| `includes`                                     | List of include dirs needed to link against the external library.                                                                                 |
| `target`                                       | one of either `host` or `target`. Whether the library is built for host or target.                                                                |
| `defines`                                      | List of strings; default is `[]`<br>Defines that are included in the local module, and all modules that depend upon it. (Including transitively.) |     |
| [`linkopts`](properties/linkopts.md)           | List of strings; default is `[]`<br>List of additional flags to the linker command.                                                               |
