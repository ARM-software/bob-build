Compiling Code
================

## My first executable

A self contained executable might simply be defined as follows.

```
bob_binary {
    name: "less",
    srcs: [
        "src/less.c",
        "src/helper.c",
    ],
    cflags: [
        "-Wall",
        "-pedantic",
        "-DDEBUG=1",
    ],
    local_include_dirs: ["include"],
    ldflags: ["-pthread"],
    ldlibs: ["-lncurses"],
}
```

This creates the executable `less` from the source files `src/less.c`
and `src/helper.c`. The text `bob_binary` is the Bob identifier for a
module type that creates executables.

When compiling these files:

 * the mentioned `cflags` will be passed to the C compiler (setting
   the warning level and defining the macro DEBUG to 1). If you
   compile C and C++ sources, then `cflags` will be applied to both C
   and C++, `conlyflags` will just be applied to C, and `cxxflags`
   will just be applied to C++.

 * the mentioned `local_include_dirs` will be passed to the C compiler
   (prefixed by '-I') to set the header search path. These directories
   are relative to the file that defines the `less` module. If you
   need to include a path from outside your source directory, use
   `include_dirs` with absolute paths instead.

When linking `less`:

 * the mentioned `ldflags` will be passed to the linker. In this case
   we're telling the linker that we will be using pthreads.

 * the mentioned `ldlibs` will be passed to the linker to identify
   system libraries to link against.

The only mandatory property is `name`. `srcs` can be empty - if the
linker doesn't complain this will end up as an empty executable. Other
properties only need to be listed if you need them.

## Code reuse

Libraries are the primary way to re-use C and C++ code. When code uses
a library it must be linked to that library. There are two types of
libraries and they are linked in different ways.

* static linking using archives (`.a` files)

* dynamic linking using shared objects (`.so` files)

Archives are simply indexed collections of object (`.o`) files.
Linking to an archive behaves like all the objects within the archive
were mentioned on the link command line. This means if two executables
statically link against the same library, each binary has a full copy
of all the code in the archive.

Shared objects are more like executables. When linking to a shared
object, metadata is added to the output file about the shared object
being linked. The runtime linker uses this metadata to find the shared
object, allowing executables to call functions in the shared library.

There are pros and cons for static vs dynamic linking. Briefly:

* statically linked executables are self contained, and do not rely on
  libraries on the users' system.

* where the same library is statically linked into multiple executables,
  more disk space will be used.

* propagating fixes to deployed systems is simpler with shared
  libraries. You just need to deploy the new shared library, and all
  callers will run the updated code. With static linking you need to
  recompile all users of the library.

In a deployed system most things will be dynamically linked. Static
linking is usually just used to manage small libraries within a code
base.

In Bob you create an archive using the `bob_static_library` module
type, and a shared object by using the `bob_shared_library` module
type. Both module types allow the same properties that are used by
`bob_binary`. Note that there are some properties that only work with
certain module types - we'll come to that later.

```
bob_binary {
    name: "less",
    srcs: ["src/less.c"],
    local_include_dirs: ["include"],
    cflags: ["-DDEBUG=1"],
    static_libs: ["libhelper"],
}

bob_static_library {
    name: "libhelper",
    srcs: ["src/helper.c"],
    local_include_dirs: ["include"],
    cflags: ["-DDEBUG=1"],
}
```

Here the executable `less` uses static linking. `helper.c` is put into
an archive. The `static_libs` property lists the `bob_static_library`
modules that need to be statically linked.

```
bob_binary {
    name: "less",
    srcs: ["src/less.c"],
    local_include_dirs: ["include"],
    cflags: ["-DDEBUG=1"],
    shared_libs: ["libhelper"],
}

bob_shared_library {
    name: "libhelper",
    srcs: ["src/helper.c"],
    local_include_dirs: ["include"],
    cflags: ["-DDEBUG=1"],
}
```

Here the executable `less` uses dynamic linking. The `shared_libs`
property lists the `bob_shared_library` modules that need to be
dynamically linked.

## Host and Target Libraries

In the simplest build setup the machine doing the compilation is
expected to run the output of the compiler. This is known as a native
build.

In some situations, rather than running on the build machine, you want
the output to run on another device (for example a phone). This is
known as cross compiling. Cross-compilation is commonly used when
bringing up a platform. It's also used if the platform isn't powerful
enough to compile code.

Bob supports cross compilation. The machine doing the build is known
as the host platform. The machine that will run the output is known as
the target platform.

When defining libraries and executables in build definitions, host and
target-specific definitions can be used.

```
bob_binary {
    name: "less",
    srcs: [
        "src/less.c",
        "src/helper.c",
    ],
    cflags: [
        "-Wall",
        "-pedantic",
        "-DDEBUG=1",
    ],
    local_include_dirs: ["include"],
    ldflags: ["-pthread"],
    ldlibs: ["-lncurses"],

    host: {
        local_include_dirs: ["include/host"],
        cflags: ["-DHOST=1"],
    },
    target: {
        local_include_dirs: ["include/target"],
        cflags: ["-DTARGET=1"],
    },
    host_supported: true,
    target_supported: true,
}
```

Properties within the `host` section are only applied to host builds of
`less`. Similarly properties in the `target` section are only applied to
target builds.

The `host_supported` and `target_supported` properties indicate
whether the library supports being built for host and target
respectively. If not specified, `target_supported` defaults to `true`
and `host_supported` defaults to `false`.

Bob doesn't prescribe what happens in a native build. We suggest that
in a native build the machine is the `target`.

### GNU Automake Convention

The Automake naming convention for cross compiling is different to
Bob's, and is based around the Canadian cross compile use case. This
is where you want to cross compile a cross compiler: you are building
a compiler on machine A, to run on machine B and generate code for
machine C.

Automake refers to the machine doing the build as the build platform, the
machine that will run the resultant compiler as the host platform, and
the machine that the compiler creates output for as the target
architecture.

|Automake configure option|Automake description|Bob platform|
|---|---|---|
|--build|Build platform|host|
|--host|Host platform|target|
|--target|Target architecture|n/a. If required this is a configuration option.|
