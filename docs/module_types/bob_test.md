# `bob_test`

> ⚠ Warning, this target is experimental & the attributes/interface are likely to keep changing. ⚠

```bp
bob_test {
    name, srcs, hdrs, copts, deps, tags, linkopts
}
```

Indicates a test binary.

On Linux this module behaves like an executable.
On Android this module generates test targets.

## Properties

|                                                |                                                                                                       |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                                                                                      |
| [`srcs`](properties/strict_properties.md)      | List of sources; default is `[]`<br>Supports glob patterns.                                           |
| `copts`                                        | List of strings; default is `[]`<br>This options are included as cflags in the compile/link commands. |
| `deps`                                         | List of targets; default is `[]`<br>The list of other libraries to be linked in to the binary target. |
| [`tags`](properties/common_properties.md#tags) | List of strings; default is `[]`                                                                      |
| [`linkopts`](properties/linkopts.md)           | List of strings; default is `[]`<br>List of additional flags to the linker command.                   |
