# `bob_filegroup`

```bp
bob_filegroup {
    name, srcs
}

```

This target lists a collection of source files that can be re-used in other targets. It exists
to enforce no relative uplinks and to closer align to Bazel.

Supports:

- [features](../features.md)
- [Gazelle generation](../../gazelle/README.md)

## Properties

|                                                |                                  |
| ---------------------------------------------- | -------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                 |
| [`srcs`](properties/strict_properties.md)      | List of sources; default is `[]` |
