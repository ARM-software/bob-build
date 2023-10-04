# Module: bob_resource

> This is a legacy target will not be supported by the [Gazelle plugin](../../gazelle/README.md).

```bp
bob_resource {
    name, srcs, exclude_srcs, enabled, build_by_default, add_to_alias, install_group, install_deps, relative_install_path, post_install_tool, post_install_cmd, post_install_args, tags, owner,
}
```

This target identifies files in the source tree which should be copied to
the installation directory, e.g. files which the project may
need while executing.

This will reference an `bob_install_group` so it gets copied to an appropriate location
relative to the binaries.

For the Android.bp backend, the `install_path` set in the
`bob_install_group` must be prefixed by a known string to select an
appropriate directory. Currently `data`, `firmware`, `etc`, `bin` and
`tests` are supported. The `owner` property also influences which
partition the files will be installed.

Supports:

- [features](../features.md)

## Properties

|                                                                          |                                                                                                                                       |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name)                           | String; required                                                                                                                      |
| [`srcs`](properties/strict_properties.md)                                | List of sources; default is `[]`<br>Source files to copy to the installation directory.patterns.                                      |
| `exclude_srcs`                                                           | List of exclude patterns; default is `[]`<br> Files to be removed from `srcs`.<br>Supports wildcards, with the same caveat as `srcs`. |
| `add_to_alias`                                                           | Target; default is `none`<br>Allows this alias to add itself to another alias.<br>Should refer to existing `bob_alias`.               |
| [`owner`](properties/legacy_properties.md#owner)                         | String; default is `none`; **deprecated**<br> If set, then the module is considered proprietary.                                      |
| [`tags`](properties/common_properties.md#tags)                           | List of strings; default is `[]`                                                                                                      |
| [`enabled`](properties/common_properties.md#enabled)                     | Boolean; default is `true`.                                                                                                           |
| `build_by_default`                                                       | Boolean; default is `true`<br>Whether it is built by default in a build with no targets requested.                                    |
| `add_to_alias`                                                           | Target; default is `none`<br>Allows this alias to add itself to another alias.<br>Should refer to existing `bob_alias`.               |
| [`install_group`](properties/legacy_properties.md#install_group)         | Target; default is `none`<br>Module name of a `bob_install_group` specifying an installation directory.                               |
| [`install_deps`](properties/legacy_properties.md#install_deps)           | List of targets; default is `[]`<br>Other modules which must be installed.                                                            |
| `relative_install_path`                                                  | String; default is `none`<br>Path to install to, relative to the install_group's path.                                                |
| `post_install_tool`                                                      | String <br>Script used during post install. Not supported on Android.                                                                 |
| [`post_install_cmd`](properties/legacy_properties.md#post_install_cmd)   | String; default is `none`<br>Command to execute on file(s) after they are installed.                                                  |
| [`post_install_args`](properties/legacy_properties.md#post_install_args) | List of strings; default is `[]`<br>Arguments to insert into `post_install_cmd`.                                                      |
