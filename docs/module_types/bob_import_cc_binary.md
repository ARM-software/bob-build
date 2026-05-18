# Module: bob_import_cc_binary

> Warning, this target is experimental and the attributes/interface are likely to keep changing.

```bp
bob_import_cc_binary {
    name, src, target
}
```

This target makes an externally built executable available in the Bob build graph. It does not expose headers, defines, or linker flags; it only provides the binary artifact to other rules.

## Properties

|                                                |                                                                                       |
| ---------------------------------------------- | ------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name) | String; required                                                                      |
| `src`                                          | Path of the built executable.                                                         |
| `target`                                       | One of either `host` or `target`. Whether the executable is built for host or target. |
