Using Libraries not Compiled by Bob
===================================

## System Libraries

System libraries are expected to be available on the target
platform. In order to compile, the system libraries also have to be
available on the development system together with any headers.

To use system libraries, use `ldlibs` to link against the library, and
`include_dirs` to add the directory containing the library headers to
the search path.

If `pkg-config` has been setup for the system library, then it can be
used to retrieve the appropriate settings for `ldlibs`, `ldflags` and
`cflags` (rather than `include_dirs`). This bulk of the work is done
in the config system, and in the build definitions we just reference
the relevant configuration.

The following example sets up the config variables required to enable
the default `host_explore.py` script to fill in information from
`pkg-config` for `libdrm`.

```
# Mconfig

config PKG_CONFIG
    bool "Enable use of pkg-config"
    default y

config PKG_CONFIG_FLAGS
    string "pkg-config flags"
    depends on PKG_CONFIG

config PKG_CONFIG_PACKAGES
    string "Packages"
    depends on PKG_CONFIG
    default "libdrm"

config PKG_CONFIG_SYSROOT_DIR
    depends on PKG_CONFIG
    string "PKG_CONFIG_SYSROOT_DIR"

config PKG_CONFIG_PATH
    string "PKG_CONFIG_PATH"
    depends on PKG_CONFIG

config LIBDRM_CFLAGS
    string "libdrm cflags"

config LIBDRM_LDFLAGS
    string "libdrm ldflags"

config LIBDRM_LDLIBS
    string "libdrm ldlibs"


// build.bp
bob_binary {
    name: "drm_test",
    srcs: ["drm_test.c"],

    cflags: ["{{.libdrm_cflags}}"],
    ldflags: ["{{.libdrm_ldflags}}"],
    ldlibs: ["{{.libdrm_ldlibs}}"],
}
```

If the library is used in a number of places you might consider making
use of `export_*` to reduce repetition:

```
bob_static_library {
    name: "libdrm_wrapper",
    srcs: [],

    export_cflags: ["{{.libdrm_cflags}}"],
    export_ldflags: ["{{.libdrm_ldflags}}"],
    ldlibs: ["{{.libdrm_ldlibs}}"],
}

bob_binary {
    name: "drm_test",
    srcs: ["drm_test.c"],

    static_libs: ["libdrm_wrapper"],
}
```

If you need to use a particular `pkg-config` binary, then use the config
`PKG_CONFIG_BINARY` to specify it.

## Android libraries

When a Bob project is built as part of Android, the project may need
to refer to Android libraries. For this, Bob has the following module
types: `bob_external_shared_library`, `bob_external_static_library`,
`bob_external_header_library`. These simply make Bob aware of the
library name, so that it can be used wherever the equivalent
non-external module types are used.

```
bob_external_static_library {
    name: "libdrm",
}

bob_binary {
    name: "drm_test",
    srcs: ["drm_test.c"],

    static_libs: ["libdrm"],
}
```

Note: the `bob_external_*` module type may be extended in the future
to support `export_*`. This would allow them to be used for system
libraries that we pick up with `pkg-config`, like in the last example
of the previous section.
