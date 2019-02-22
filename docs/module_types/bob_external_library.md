Module: bob_external_header_library, bob_external_shared_library, bob_external_static_library
=============================================================================================

External libraries are a method of linking with Android libraries defined
outside of Bob.

## Full specification of `bob_external_[header|shared|static]_library` properties

The external library types only support a single property, `name`, which should
match the name of the corresponding Android library.

```bp
bob_external_static_library {
    name: "libname",
}
```
