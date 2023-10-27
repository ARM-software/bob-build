# Additional linker options

`linkopts` property allows to provide additional linker options. However there is no any filtering or checking for them - they are passed as is.

Depending on the backend, linker options may behave a bit different thus need to be used carefully. Check below how they are applied for particular backends.
If needed, they can be separated with [features](../../features.md).

## Example

```bp
bob_library {
    name: "libname",
    srcs: ["libname.cpp"],

    builder_android_bp: {
        linkopts: [
            "-u fake_android",
        ],
    },

    builder_ninja: {
        linkopts: [
            "-u fake_linux",
        ],
    },
}
```

## Linux Backend

`linkopts` options are appended to the end of the linker command in the order specified.

## Android backend

`linkopts` options are directly mapped to Android module's `ldflags` property.
