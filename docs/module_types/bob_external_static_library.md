# `bob_external_static_library`

```bp
bob_external_static_library {
    name, export_cflags, export_ldflags, ldlibs,
}
```

External libraries are a method of linking with Android libraries defined
outside of Bob.

## Properties

|                                                                    |                                                                                                       |
| ------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name)                     | String; required<br>The name should correspond to the Android library.                                |
| [`ldlibs`](properties/legacy_properties.md#ldlibs)                 | List of strings; default is `[]`<br>Linker flags required to link to the necessary system libraries.  |
| [`export_cflags`](properties/legacy_properties.md#export_cflags)   | List of strings; default is `[]`<br>C/C++ flags exported to modules which depend on the current one.  |
| [`export_ldflags`](properties/legacy_properties.md#export_ldflags) | List of strings; default is `[]`<br>Linker flags exported to modules which depend on the current one. |
