# bob_defaults

```bp
bob_defaults {
    name, srcs, exclude_srcs, enabled, build_by_default, add_to_alias, defaults, target_supported, target, host_supported, host, out, cflags, export_cflags, cxxflags, asflags, conlyflags, ldflags, export_ldflags, static_libs, shared_libs, reexport_libs, whole_static_libs, ldlibs, generated_headers, generated_sources, generated_deps, tags, owner, strip, include_dirs, local_include_dirs, export_local_include_dirs, export_include_dirs, build_wrapper, forwarding_shlib, kbuild_options, extra_symbols, make_args, kernel_dir, kernel_cross_compile, kernel_cc, kernel_hostcc, kernel_clang_triple, install_group, install_deps, relative_install_path, debug_info, post_install_tool, post_install_cmd, post_install_args, tags
}
```

Defaults are used to share common settings between modules.

Supports:

- [features](../features.md)
- [defaults](./bob_defaults.md) (recursive defaults are supported)

## Attributes

`bob_defaults` supports the same attributes as its consuming module. Refer to each module for detailed attribute documentation.

## Examples

### Simple

Here, the `-lncurses` flag is used with linking the `less` binary,
because it has been propagated through the default.

```bp
bob_defaults {
    name: "common_libs",
    ldlibs: ["-lncurses"],
}

bob_binary {
    name: "less",
    defaults: ["common_libs"],
    srcs: ["src/less.c"],
}
```

### Nested Defaults

Defaults can be hierarchical by including other defaults. Here, the
`my_unit_test` binary will inherit both the `-Wall` and `-UNDEBUG` flags.

```bp
bob_defaults {
    name: "project_wide",
    cflags: ["-Wall"],
}

bob_defaults {
    name: "unit_test_defaults",
    defaults: ["project_wide"],
    cflags: ["-UNDEBUG"],
}

bob_binary {
    name: "my_unit_test",
    defaults: ["unit_test_defaults"],
    srcs: ["unit_test.c"],
}
```
