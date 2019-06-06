Build Wrappers
==============

In some circumstances you need to be able to intercept all calls to
the compiler and linker. Examples of this are `ccache` and
`distcc`.

`ccache` caches previous results of running the compiler, indexed by
the hash of the pre-processed source code. `ccache` detects when a
subsequent run should produce the same output, and instead of
executing just uses the previous output.

`distcc` farms out calls to the compiler to a compilation server
farm. This means compilation can be performed across many machines
instead of just the local machine.

These tools may be set up to replace the compiler binaries on the PATH
and they would call the location of the real compilers; or you just
prefix the compiler call with `ccache` (or `distcc`). In the former
situation, Bob does not need to do anything. In the latter you need to
use the `build_wrapper` property.

```
bob_binary {
    name: "less",
    srcs: ["less.c"],
    build_wrapper: "ccache",
}
```

The build wrapper is not limited to these two binaries. Arbitrary
scripts can be used, as long as they supply the output expected of the
compiler.
