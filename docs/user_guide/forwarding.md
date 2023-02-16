# Forwarding Libraries

Consider a project implementing multiple APIs which share a lot of
common code in their implementations. This might be done as follows:

```
bob_shared_library {
    name: "libcommon",
    srcs: ["libcommon/common.c"],
    export_local_include_dirs: ["libcommon/include"],
}

bob_shared_library {
    name: "liba_api",
    srcs: ["a/a.c"],
    shared_libs: ["libcommon"],
}

bob_shared_library {
    name: "libb_api",
    srcs: ["b/b.c"],
    shared_libs: ["libcommon"],
}

bob_shared_library {
    name: "libc_api",
    srcs: ["c/c.c"],
    shared_libs: ["libcommon"],
}
```

However you find that requirements are changing rapidly and the
libcommon API is very unstable. This causes problems in your
deployments as users are having trouble keeping the right versions of
the libraries in place.

What you want to do is compile a single monolithic shared library that
implements all the APIs, and have small libraries that just forward
all calls to the one library. Bob calls these small libraries
forwarding libraries, as they just forward calls to another
library. They may sometimes be referred to as stub libraries. You can
declare a forwarding shared library in Bob by using the `forwarding_shlib`
property on `bob_shared_library`s.

```
bob_shared_library {
    name: "libcommon",
    srcs: [
        "libcommon/common.c",
        "a/a.c",
        "b/b.c",
        "c/c.c"
    ],
    local_include_dirs: ["libcommon/include"],
}

bob_shared_library {
    name: "liba_api",
    srcs: [],
    shared_libs: ["libcommon"],

    forwarding_lib: true,
}

bob_shared_library {
    name: "libb_api",
    srcs: [],
    shared_libs: ["libcommon"],

    forwarding_lib: true,
}

bob_shared_library {
    name: "libc_api",
    srcs: [],
    shared_libs: ["libcommon"],

    forwarding_lib: true,
}
```

In its simplest form a forwarding shared library can be implemented
as a symlink. A downside of the symlink implementation is that you can
only have one target library, and you can't have different library
version information on each forwarding library.

Bob implements forwarding libraries by adding the `DT_NEEDED` symbol
to the library by using the linker's `--no-as-needed` flag. Things that
link to the forwarding library need to use `--copy-dt-needed-entries`
in addition to `--no-as-needed`. This restricts forwarding libraries
to the BFD linker, as the more recent linkers do not support
`--copy-dt-needed-entries`.

Our recommendation is not to use forwarding libraries, and come up
with stable interfaces for the common library.

## Some History

- Previously Linux distributions were in a position that dependencies
  were badly specified, the usual breakage happening where a binary
  assumed `lib1` depended on `lib2` and linked against both. Then the
  dependencies of `lib1` changed and swapped `lib2` for `lib3`, and
  the system would not work after `lib1` was updated.

- The community have settled on binaries (and shared libraries)
  explicitly depending on the libraries that resolve the symbols that
  they use. So nowadays the binary will only depend on `lib1`.

- Originally the default BFD linker behaviour was equivalent to
  `--copy-dt-needed-entries --no-as-needed`.

- `--copy-dt-needed-entries` has the following behaviour:

  - The linker will recursively search libraries to resolve symbols. A
    binary depending on `lib1` that depends on `lib2` will pick up
    symbols from `lib2`.

  - A binary depending on `lib1` that depends on `lib2` gets a `DT_NEEDED lib2.so` entry.

- `--no-as-needed` has the following behaviour:

  - add `DT_NEEDED` for all libraries on the command line even if no
    symbols from the library are used.

- The default linker behaviour changed to `--no-copy-dt-needed-entries --as-needed`. This avoids adding unnecessary library dependencies in
  the system.

- Neither `lld` nor `gold` support `--copy-dt-needed-entries`. They do
  support `--no-as-needed`.

- To implement forwarding libraries:

  - The link of the forwarding library needs to use `--no-as-needed` so
    that `liba.so` retains `DT_NEEDED libcommon.so`

  - The link of the binary using the forwarding library needs to use
    `--no-as-needed` so that the binary retains `DT_NEEDED liba.so`. It also needs `--copy-dt-needed-entries` so that the
    linker searches `libcommon.so` for the symbols needed by the
    binary. This has the unfortunate effect of also adding `DT_NEEDED libcommon.so` to the binary.
