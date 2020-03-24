Advanced Library Use
====================

This section covers a few more advanced use cases associated with
libraries.

## Library compilation and link requirements

In order to use a library the calling code may have to be compiled and
linked with particular options. At a minimum it needs to specify the
path where the library headers can be found.

Generally this means setting up `cflags`, `local_include_dirs`,
`ldflags` and `ldlibs` on each caller. When you have many users of a
library it can be tedious to keep all the callers correct. Bob
provides the `export_*` properties to help manage this.

```
bob_binary {
    name: "less",
    srcs: ["src/less.c"],
    shared_libs: ["libhelper"],
}

bob_binary {
    name: "gzip",
    srcs: ["src/gzip.c"],
    shared_libs: ["libhelper"],
}

bob_shared_library {
    name: "libhelper",
    srcs: ["helper/helper.c"],
    cflags: ["-DDEBUG=1"],
    local_include_dirs: ["helper/private/include"],
    export_local_include_dirs: ["helper/include"],
    export_cflags: ["-DHELP=1"],
}
```

In the above, both the `less` and `gzip` compiles will have `-DHELP=1
-Ihelper/include` because each binary is using `libhelper` which has
exported these properties. The `libhelper` compile will also have
`-DDEBUG=1 -Ihelper/private/include` specified, but since these aren't
exported, neither `less` nor `gzip` would get it. The non-`export`
versions only affect the current module.

`export_local_include_dirs` identifies where the external header(s)
for the library live. If you keep include directories specific to each
library and use `export_local_include_dirs` then the build definition
will keep track of where libraries are used and simplify dependency
management.

The requirement to use `export_cflags` usually comes from the library
referencing some macros (`#defines`) in its external headers. This is
not recommended; the external headers define a library's interface,
which should remain constant regardless of configuration. Maintaining
a constant interface allows the library to be updated without forcing
its users to rebuild.

A library may call code from other libraries. To ensure that the
executable (or shared library) links against the right thing, use
`static_libs`, `shared_libs` and `ldlibs`. These effectively say
"I'm using this library, make sure it is added to your list of
(static|shared|system) libraries".

For completeness, `export_ldflags` is also supported. This propagates
linker flags to modules that link against this library. You can use
this if the library requires particular link flags to be used, however
this affects the other libraries being linked so should generally be
avoided. One use case for this is `-pthread`.

`export_cflags` and `export_include_dirs` only affect modules that
directly use the library doing the export. Sometimes they need to
propagate through a number of libraries. This is necessary where the
external interface of a library includes files from the libraries that
it uses. To allow this add `reexport_libs` in the intermediate
libraries.

```
bob_binary {
    name: "tar",
    srcs: ["tar/tar.c"],
    static_libs: ["libcompression"],
}

bob_static_library {
    name: "libcompression",
    srcs: ["compression/compression.c"],
    export_local_include_dirs: ["compression/include"],
    static_libs: ["libgzip"],

    // One of the libcompression's external headers includes a gzip header
    reexport_libs: ["libgzip"],
}

bob_static_library {
    name: "libgzip",
    srcs: ["libgzip/gzip.c"],
    export_local_include_dirs: ["gzip/include"],
}
```

On static libraries `static_libs`, `shared_libs`, `ldlibs`, and `export_ldflags`
always propagate to the nearest module doing the link i.e. the nearest
`bob_binary` or `bob_shared_library`.

## Static Library Encapsulation

When code uses a library there may be an expectation that nothing else
needs to know about the library. In the following example we have a
compression library that encapsulates multiple file compression
algorithms.

```
bob_binary {
    name: "tar",
    srcs: ["tar.c"],
    static_libs: ["libhelpers"],
}

bob_binary {
    name: "untar",
    srcs: ["untar.c"],
    static_libs: ["libhelpers"],
}

bob_static_library {
    name: "libhelpers",
    static_libs: [
        "libcompression",
        "libsha1",
        "libutf8",
    ],
}

bob_static_library {
    name: "libcompression",
    srcs: [
        "compress.c",
        "inflate.c",
    ],
    static_libs: [
        "libgzip",
        "libbzip",
        "liblzma",
    ],
}
```

The above definition works and is usable. However it allows another
module to use `libgzip` when maybe it should really be using the
wrapper library:

```
bob_binary {
    name: "gzip",
    static_libs: [
        "libhelpers",
        "libgzip",
    ],
}
```

The encapsulating library can use `whole_static_libs` to indicate it
is encapsulating these libraries. Bob will raise an error if an
executable (or shared library) tries to link both the encapsulating
library and the encapsulated libraries.

```
bob_binary {
    name: "tar",
    srcs: ["tar.c"],
    static_libs: ["libhelpers"],
}

bob_binary {
    name: "untar",
    srcs: ["untar.c"],
    static_libs: ["libhelpers"],
}

bob_static_library {
    name: "libhelpers",
    whole_static_libs: [
        "libcompression",
        "libsha1",
        "libutf8",
    ],
}

bob_static_library {
    name: "libcompression",
    srcs: [
        "compress.c",
        "inflate.c",
    ],
    whole_static_libs: [
        "libgzip",
        "libbzip",
        "liblzma",
    ],
}
```

## Link What You Use

When `static_libs` and `shared_libs` are used extensively on static
libraries (where they are propagated) you may find that builds happen
to work because the exports from low level libraries satisfy the
requirements of higher level libraries.

Although this works, avoid relying on this. This is an analogous
situation to C/C++ header usage, and the advice is the same. Instead
of "Include What You Use", "Link What You Use" - for every (external)
symbol in the `srcs` of your module the library that supplies the
symbol must be listed in either `static_libs`, `shared_libs` or
`whole_static_libs`. Symbols include function calls and global
variable access. Note that for enumeration and macro values you can
get away with just including headers, and not linking - however it's
simpler to specify the link and pick up the header path from the
library.

## Circular dependencies

Libraries can be written so that they are mutually dependent, or have
a circular dependency on each other. A mutual dependency is just the
case of a circular dependency between two libraries. The following is
a simple example where someone has decided to add some debugging code
into their `print()` function (please don't do this).

```C
/* libdebug */
void debug(const char *msg) {
    print(STDERR, "DEBUG: ");
    print(STDERR, msg);
}
```

```C
/* libstream */
void print(FILE *f, const char msg) {
    if (f != STDERR) {
        debug("About to print ");
        debug(msg);
    }

    printf(STDOUT, msg);
}
```

In this case there is a mutual dependency between `libdebug` and
`libstream`. When you try to link these libraries, the linker will
tell you that either the symbol `print` or `debug` cannot be resolved.

Note that there isn't a problem if these functions are just in object
files being linked into a single library.

Always try to eliminate the circular dependency first, as this is a
sign of a design problem. Here `libdebug` uses `libstream` for output,
therefore `libstream` cannot use `libdebug`.

This may not always be possible. Bob can handle the situation where
the circular dependency just involves static libraries. In this case,
encapsulate all the static libraries in the chain in a new static
library using the `whole_static_libs` property. This has the effect
that this just looks like multiple `.o` files.

```
bob_static_library {
    name: "libhelper",
    whole_static_libs: [
        "libdebug",
        "libprint",
    ],
}
```

## Supporting different generations of compiler flags

Sometimes a library needs to be compatible with a whole range of
compiler versions that might have different behaviours between each
generation. One common usage is to avoid new warnings being introduced
by more modern compilers.

In that case you can use the template `{{add_if_supported
"<compiler_flag>"}}` in the `cflags`, `cxxflags` or `conlyflags` to
instruct Bob to add the given compiler flag to the compiler arguments,
but only if the compiler recognises that flag. If the compiler doesn't
recognise the `<compiler_flag>` it will be silently discarded from
`cflags`, `cxxflags` and `conlyflags`.

## Retaining API functions in shared libraries

The linker will typically remove unused code from the output file. For
executables, the linker knows to look for code that cannot be reached
from `main()` (or what is specified by `--entry`). For shared
libraries there isn't an equivalent entry point.

Bob currently lacks a general way to specify the entrypoints of a
shared library. Instead use `whole_static_libs` to specify that all
the external symbols in the mentioned static libraries need to be
retained in the shared library.

```
bob_static_library {
    name: "libcompapi",
    srcs: "api.c",
    static_libs: [
        "libgzip",
        "libbzip",
        "liblzma",
    ],
}

bob_static_library {
    name: "libhelpers",
    srcs: "helpers.c"
    static_libs: [
         "libsha1",
         "libutf8",
    ],
}

bob_shared_library {
    name: "libcompression",

    // symbols in file1.c would be kept unless -ffunction-sections
    // -fdata-sections --gc-sections are specified
    srcs: ["file1.c"],

    // symbols in api.c are kept
    // we can discard symbols from libgzip, libbzip and liblzma
    whole_static_libs: ["libcompapi"],

    // we can discard symbols from libhelpers and its libraries
    static_libs: ["libhelpers"],
}
```

Note: we expect to improve how this is done so that the linker can be
more aggressive in dead code removal.

## Stripping libraries

Information is normally inserted into libraries to help debug issues
and produce readable stack traces.

The information can significantly increase the size of the library.
This primarily affects the storage requirements for the library.

Since this information is not needed during execution you may want to
remove it from public releases. The `strip` property achieves this.

```
bob_shared_library {
    name: "libcompression",
    srcs: ["file1.c"],

    // RELEASE is a configuration
    release: {
        strip: true,
    },
}
```

Stripping all information can hinder debugging issues that occur in
the field. To mitigate this, the debug information can be kept in a
separate file which does not need to be released. Use the `debug_info`
property to indicate that separate debug information is desired. The
property must reference an install group to indicate where to save the
debug information. This can be used independently of `strip` to
separate out the debug information in normal builds. This only affects
non-Android builds.

```
bob_install_group {
    name: "IG_debug",
    install_path: "install/debug",
}

bob_shared_library {
    name: "libcompression",
    srcs: ["file1.c"],
    debug_info: "IG_debug",

    // RELEASE is a configuration
    release: {
        strip: true,
    },
}
```

Note that if `install_path` of a `bob_install_group` used for debug
information is empty "", the debug information files are placed
alongside the library in question.

When `install_path` is set to a directory as normal, GDB will expect
one of a few layouts, see [GDB documentation](https://sourceware.org/gdb/onlinedocs/gdb/Separate-Debug-Files.html).
The helper script `scripts/move_debug_files.py` can be used to move
the debug files into a layout based on build IDs. This may need
`-Wl,--build-id` to be passed to the linker.
