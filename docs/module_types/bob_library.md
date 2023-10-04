# Module: bob_library

> ⚠ Warning, this target is experimental & the attributes/interface are likely to keep changing. ⚠

```bp
bob_library {
    name, srcs, hdrs, copts, local_defines, defines, deps
}
```

This target replaces `bob_static_library` & `bob_shared_library` to mimic Bazel and Soong in having a single library rule and being context aware to understand if a library should be statically built or dynamically.

## Properties

|                                                |                                                                                                                                                   |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                                                                                                                                  |
| [`srcs`](properties/strict_properties.md)      | List of sources; default is `[]`<br>Supports glob patterns.                                                                                       |
| `hdrs`                                         | List of sources; default is `[]`<br>Headers that are a part of the library.                                                                       |
| `defines`                                      | List of strings; default is `[]`<br>Defines that are included in the local module, and all modules that depend upon it. (Including transitively.) |
| `local_defines`                                | List of strings; default is `[]`<br>Defines that are local to the module and are not added to modules that depend upon this.                      |
| `copts`                                        | List of strings; default is `[]`<br>This options are included as cflags in the compile/link commands.                                             |
| `deps`                                         | List of targets; default is `[]`<br>The list of other libraries to be linked in to the binary target.                                             |
