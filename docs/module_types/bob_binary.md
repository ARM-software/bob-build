# `bob_binary`

> This is a legacy target and will not be supported by the [Gazelle plugin](../../gazelle/README.md)

```bp
bob_binary {
    name, srcs, exclude_srcs, enabled, build_by_default, add_to_alias, defaults, target_supported, target, host_supported, host, out, cflags, cxxflags, asflags, conlyflags, ldflags, ldlibs, static_libs, shared_libs, generated_headers, generated_sources, generated_deps, tags, owner, strip, include_dirs, local_include_dirs, build_wrapper, add_lib_dirs_to_rpath, install_group, install_deps, relative_install_path, debug_info, post_install_tool, post_install_cmd, post_install_args, version_script
}
```

Target is an executable.

Supports:

- [features](../features.md)
- [defaults](./bob_defaults.md)

## Properties

|                                                                                  |                                                                                                                                                          |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name)                                   | String; required                                                                                                                                         |
| [`srcs`](properties/common_properties.md#srcs)                                   | List of sources; default is `[]`                                                                                                                         |
| `exclude_srcs`                                                                   | List of exclude patterns; default is `[]`<br> Files to be removed from `srcs`.<br>Supports wildcards, with the same caveat as `srcs`.                    |
| [`enabled`](properties/common_properties.md#enabled)                             | Boolean; default is `true`.                                                                                                                              |
| `build_by_default`                                                               | Boolean; default is `true`<br>Whether it is built by default in a build with no targets requested.                                                       |
| `add_to_alias`                                                                   | Target; default is `none`<br>Allows this alias to add itself to another alias.<br>Should refer to existing `bob_alias`.                                  |
| [`defaults`](properties/legacy_properties.md#defaults)                           | List of [`bob_defaults`](bob_defaults.md); default is `[]`                                                                                               |
| [`target_supported`](properties/common_properties.md#target_supported)           | Boolean; default is `true`.                                                                                                                              |
| [`target`](properties/common_properties.md#target)                               | Property map; default is `{}`.                                                                                                                           |
| [`host_supported`](<(properties/common_properties.md#host_supported)>)           | Boolean; default is `false`.                                                                                                                             |
| [`host`](<(properties/common_properties.md#host)>)                               | Property map; default is `{}`.                                                                                                                           |
| `out`                                                                            | String;<br>Alternate output name, used for the file name and Android rules.                                                                              |
| [`cflags`](properties/legacy_properties.md#cflags)                               | List of strings; default is `[]`<br>Flags used for C/C++ compilation.                                                                                    |
| `conlyflags`                                                                     | List of strings; default is `[]`<br>Flags used for C compilation.<br>See [`cflags`](properties/legacy_properties.md#cflags)                              |
| `cxxflags`                                                                       | List of strings; default is `[]`<br>Flags used for C++ compilation.<br>See [`cflags`](properties/legacy_properties.md#cflags)                            |
| [`asflags`](properties/legacy_properties.md#asflags)                             | List of strings; default is `[]`<br>Flags used for assembly compilation.                                                                                 |
| [`ldflags`](properties/legacy_properties.md#ldflags)                             | List of strings; default is `[]`<br>Flags used for linking.                                                                                              |
| [`ldlibs`](properties/legacy_properties.md#ldlibs)                               | List of strings; default is `[]`<br>Linker flags required to link to the necessary system libraries.                                                     |
| [`static_libs`](properties/legacy_properties.md#static_libs)                     | List of targets; default is `[]`<br>The list of static lib modules that this library depends on.                                                         |
| [`shared_libs`](properties/legacy_properties.md#shared_libs)                     | List of targets; default is `[]`<br>                                                                                                                     |
| [`generated_headers`](properties/legacy_properties.md#generated_headers)         | List of targets; default is `[]`<br>                                                                                                                     |
| [`generated_sources`](properties/legacy_properties.md#generated_sources)         | List of targets; default is `[]`<br>                                                                                                                     |
| [`generated_deps`](properties/legacy_properties.md#generated_deps)               | List of targets; default is `[]`<br>                                                                                                                     |
| [`owner`](properties/legacy_properties.md#owner)                                 | String; default is `none`; **deprecated**<br> If set, then the module is considered proprietary.                                                         |
| [`strip`](properties/legacy_properties.md#strip)                                 | Boolean; default is `false`.<br> When set, strip symbols and debug information from libraries and binaries.                                              |
| [`include_dirs`](properties/legacy_properties.md#include_dirs)                   | List of strings; default is `[]`<br>A list of include directories to use. These are expected to be system headers, and will usually be an absolute path. |
| [`local_include_dirs`](properties/legacy_properties.md#local_include_dirs)       | List of strings; default is `[]`<br>A list of include directories to use. These are relative to the `build.bp` containing the module definition          |
| [`build_wrapper`](properties/legacy_properties.md#build_wrapper)                 | String; default is `none`.<br>Wrapper for all build commands.                                                                                            |
| [`add_lib_dirs_to_rpath`](properties/legacy_properties.md#add_lib_dirs_to_rpath) | Boolean; default is `false`<br>If true, the module's shared libraries' directories will be added to its DT_RUNPATH entry.                                |
| [`install_group`](properties/legacy_properties.md#install_group)                 | Target; default is `none`<br>Module name of a `bob_install_group` specifying an installation directory.                                                  |
| [`install_deps`](properties/legacy_properties.md#install_deps)                   | List of targets; default is `[]`<br>Other modules which must be installed.                                                                               |
| `relative_install_path`                                                          | String; default is `none`<br>Path to install to, relative to the install_group's path.                                                                   |
| [`debug_info`](properties/legacy_properties.md#debug_info)                       | Target; default is `none`<br>Module name of a `bob_install_group` specifying an installation directory for debug information.                            |
| `post_install_tool`                                                              | String <br>Script used during post install. Not supported on Android.                                                                                    |
| [`post_install_cmd`](properties/legacy_properties.md#post_install_cmd)           | String; default is `none`<br>Command to execute on file(s) after they are installed.                                                                     |
| [`post_install_args`](properties/legacy_properties.md#post_install_args)         | List of strings; default is `[]`<br>Arguments to insert into `post_install_cmd`.                                                                         |
| `version_script`                                                                 | Linker script used for [symbol versioning](../user_guide/libraries_2.md#markdown-header-symbol-versioning).                                              |
| [`tags`](properties/common_properties.md#tags)                                   | List of strings; default is `[]`                                                                                                                         |
