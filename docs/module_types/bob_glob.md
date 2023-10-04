# `bob_glob`

```bp
bob_glob {
    name, srcs, exclude, allow_empty,
}
```

Glob is a helper module that finds all files that match certain path patterns
and returns a list of their paths.

Supports:

- [Gazelle generation](../../gazelle/README.md)

## Properties

|                                                |                                                                                                                                                  |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| [`name`](properties/common_properties.md#name) | String; required                                                                                                                                 |
| [`srcs`](properties/strict_properties.md)      | List of sources; default is `[]`<br>Supports glob patterns.                                                                                      |
| `exclude`                                      | List of sources; default is `[]`<br>Path patterns that are relative to the current module to exclude from `srcs`.                                |
| `allow_empty`                                  | Boolean; default is `true`<br>If the `allow_empty` argument is set to `false`, the glob function will error-out if the result is the empty list. |
