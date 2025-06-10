# `bob_kernel_module`

> This is a legacy target will not be supported by the [Gazelle plugin](../../gazelle/README.md).

```bp
bob_kernel_module {
    name, srcs, exclude_srcs, enabled, build_by_default, add_to_alias, defaults, cflags, tags, include_dirs, local_include_dirs, kbuild_options, extra_symbols, make_args, kernel_dir, kernel_cross_compile, kernel_cc, kernel_hostcc, kernel_clang_triple, install_group, install_deps, relative_install_path, post_install_tool, post_install_cmd, post_install_args,
}
```

This target is a kernel module (.ko) that will be built as an
external kernel module.

A kernel module is defined using the build rule
`bob_kernel_module`. This invokes an out-of-tree kernel build for
the module in question.

To avoid cluttering the source directory, the `bob_kernel_module` will
copy all of its sources to the build directory before invoking Kbuild.
This means that _all_ files in the module directory, including
`Kbuild`, `Makefile`, and `.h` files, must be included in the
`bob_kernel_module.srcs` list.

Furthermore, the `build.bp` containing the `bob_kernel_module`
definition must be in the same directory as the main `Kbuild` file for
that module.

Supports:

- [features](../features.md)

## Properties

|                                                                            |                                                                                                                                                                                                                                                                              |
| -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name)                             | String; required                                                                                                                                                                                                                                                             |
| [`srcs`](properties/common_properties.md#srcs)                             | List of sources; default is `[]`                                                                                                                                                                                                                                             |
| `exclude_srcs`                                                             | List of exclude patterns; default is `[]`<br> Files to be removed from `srcs`.<br>Supports wildcards, with the same caveat as `srcs`.                                                                                                                                        |
| [`enabled`](properties/common_properties.md#enabled)                       | Boolean; default is `true`.                                                                                                                                                                                                                                                  |
| `build_by_default`                                                         | Boolean; default is `true`<br>Whether it is built by default in a build with no targets requested.                                                                                                                                                                           |
| `add_to_alias`                                                             | Target; default is `none`<br>Allows this alias to add itself to another alias.<br>Should refer to existing `bob_alias`.                                                                                                                                                      |
| [`defaults`](properties/legacy_properties.md#defaults)                     | List of [`bob_defaults`](bob_defaults.md); default is `[]`                                                                                                                                                                                                                   |
| [`cflags`](properties/legacy_properties.md#cflags)                         | List of strings; default is `[]`<br>Flags used for C/C++ compilation.                                                                                                                                                                                                        |
| [`tags`](properties/common_properties.md#tags)                             | List of strings; default is `[]`                                                                                                                                                                                                                                             |
| [`include_dirs`](properties/legacy_properties.md#include_dirs)             | List of strings; default is `[]`<br>A list of include directories to use. These are expected to be system headers, and will usually be an absolute path.                                                                                                                     |
| [`local_include_dirs`](properties/legacy_properties.md#local_include_dirs) | List of strings; default is `[]`<br>A list of include directories to use. These are relative to the `build.bp` containing the module definition                                                                                                                              |
| `kbuild_options`                                                           | List of strings; <br>Linux kernel config options to emulate. <br> These are passed to Kbuild in the `make` command-line, and set in the source code via `EXTRA_CFLAGS`. These should usually include the `CONFIG_` prefix, although it is possible to omit this if required. |
| `extra_symbols`                                                            | List of strings; <br>Kernel modules which this module depends on.                                                                                                                                                                                                            |
| `make_args`                                                                | List of strings; <br>Arguments to pass to kernel make invocation.                                                                                                                                                                                                            |
| `kernel_dir`                                                               | String <br>Kernel directory location. This must either be absolute or relative to the top level source directory.                                                                                                                                                            |
| `kernel_cross_compile`                                                     | String <br>Compiler prefix for kernel build.                                                                                                                                                                                                                                 |
| `kernel_cc`                                                                | String <br>Kernel target compiler.                                                                                                                                                                                                                                           |
| `kernel_hostcc`                                                            | String <br>Kernel host compiler.                                                                                                                                                                                                                                             |
| `kernel_clang_triple`                                                      | String <br>Target triple when using clang as the compiler.                                                                                                                                                                                                                   |
| [`install_group`](properties/legacy_properties.md#install_group)           | Target; default is `none`<br>Module name of a `bob_install_group` specifying an installation directory.                                                                                                                                                                      |
| [`install_deps`](properties/legacy_properties.md#install_deps)             | List of targets; default is `[]`<br>Other modules which must be installed.                                                                                                                                                                                                   |
| `relative_install_path`                                                    | String; default is `none`<br>Path to install to, relative to the install_group's path.                                                                                                                                                                                       |
| `post_install_tool`                                                        | String <br>Script used during post install. Not supported on Android.                                                                                                                                                                                                        |
| [`post_install_cmd`](properties/legacy_properties.md#post_install_cmd)     | String; default is `none`<br>Command to execute on file(s) after they are installed.                                                                                                                                                                                         |
| [`post_install_args`](properties/legacy_properties.md#post_install_args)   | List of strings; default is `[]`<br>Arguments to insert into `post_install_cmd`.                                                                                                                                                                                             |
