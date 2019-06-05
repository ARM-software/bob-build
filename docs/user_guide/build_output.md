Build Output
============

The layout of the build tree is arranged so that the compilation for
each module is independent. In general it's not necessary to know the
actual layout.

As a user you need to be able to control where executables and
libraries end up, so that they can be run in the production
environment. Conversion between the internal build tree layout and the
user specified layout is referred to as the 'install' phase.

## Install groups

An install group refers to items that we expect to end up in a common
directory. The idea is to group objects by type, and assign modules to
the appropriate groups.

The install groups for a project with cross compilation might look
something like:

```
bob_install_group {
    name: "IG_host_executables",
    install_path: "host_install/bin",
}

bob_install_group {
    name: "IG_host_libraries",
    install_path: "host_install/lib",
}

bob_install_group {
    name: "IG_executables",
    install_path: "install/bin",
}

bob_install_group {
    name: "IG_libraries"
    install_path: "install/lib",
}

bob_install_group {
    name: "IG_modules"
    install_path: "install/lib/modules",
}

bob_install_group {
    name: "IG_tests",
    install_path: "install/tests",
}

bob_install_group {
    name: "IG_testdata",
    install_path: "install/tests/data",
}

bob_install_group {
    name: "IG_configuration",
    install_path: "install/etc",
}

bob_install_group {
    name: "IG_documentation",
    install_path: "install/usr/share/doc",
}
```

Each `install_path` is referring to a directory under the build output
directory. It's recommended to set up a directory containing all the
files to be installed, so that they can be copied to the target
filesystem in one command.

## Installation properties

Each module that needs to be installed sets a few properties.

```
bob_shared_library {
    name: "libdrm",
    srcs: ["drm.c"],

    install_group: "IG_libraries",
    relative_install_path: "libdrm",
    post_install_tool: "libdrm_post.py",
    post_install_cmd: "${tool} ${out}",

    install_deps: ["formats"],
}
```

In most cases just set `install_group`, which places `libdrm.so` under
`install/lib`. `relative_install_path` allows you to specify a
subdirectory, so that it's easier to setup more complicated
heirarchies without lots of install groups. Here `libdrm.so` actually
gets copied to `install/lib/libdrm/libdrm.so`.

If you need to post-process the binary, use `post_install_cmd`. The
related `post_install_tool` will add a dependency on a script which
can be referred to in `post_install_cmd` as `${tool}`. An example
of post-processing is to strip libraries of debug information.

When installing libraries and binaries, their dependencies are also
installed. You can specify additional dependencies with
`install_deps`. For example if a test binary reads configuration from
a text file, specify the text file as an install dependency.

These install properties are available on all module types.

## Resources

`bob_resource` is a module type that identifies files in the source
tree which are copied to the installation directory without any other
processing.

```
bob_resource {
    name: "formats",
    srcs: ["formats.txt"],
    install_group: "IG_configuration",
}
```
