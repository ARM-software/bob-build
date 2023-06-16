# Module: bob_external_header_library, bob_external_shared_library, bob_external_static_library

External libraries are a method of linking with Android libraries defined
outside of Bob.

## Full specification of `bob_external_[header|shared|static]_library` properties

The `name` property should match the name of the corresponding Android library.
For detailed documentation of the attributes shown below please see [common module properties](common_module_properties.md).

```bp
bob_external_shared_library {
    name: "libname",
    export_cflags: ["..."],
    export_ldflags: ["..."],
    ldlibs: ["..."],
}
```
