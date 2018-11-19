Bob Build System
================

## Introduction

Bob is a declarative build system intended to build C/C++ software for
both Linux and Android.

Bob has a configuration system that works in a similar way to the
[Linux Kernel](https://www.kernel.org)'s
[Kconfig](https://www.kernel.org/doc/Documentation/kbuild/kconfig-language.txt).

Build definitions use a JSON-like syntax to describe the modules to
build.

Bob uses Google's [Blueprint](https://github.com/google/blueprint) to
do the heavy lifting. As such it has similarities with
[Soong](https://android.googlesource.com/platform/build/soong).

## Requirements

To use Bob you will need:
-  golang (>=1.8)
-  ninja-build (>=1.8)
-  python
-  python-ply

## License

The software is provided under the Apache 2.0 license. Contributions
to this project are accepted under the same license.

## Usage

### Setting up a project

Refer to the [documentation](docs/project_setup.md) for instructions
on how to setup a project to use Bob.

### Config file format

The config file format is simplified Kconfig, with `bool`, `int` and
`string` types.

```
config USE_LOCALES
    bool "Use Locales"
    default y

config DEFAULT_LOCALE
    string "Default Locale"
    depends on USE_LOCALES
    default "sv_SE"
```

For more information see the [documentation](docs/config_system.md).

### Build file format

The build files are very simple. There are no conditional or control
flow statements.

```
bob_defaults {
    name: "common_libs",
    ldlibs: ["-lncurses"],
}

bob_binary {
    name: "less",
    defaults: ["common_libs"],
    srcs: ["src/less.c"],

    // use_locales is a feature. When enabled in the configuration
    // src/locales.c will be compiled and linked.
    use_locales: {
        srcs: ["src/locales.c"],
        cflags: ["-DDEFAULT_LOCALE={{.default_locale}}"],
    }
}
```

For more information see the [documentation](docs/build_defs.md).

## Development

### Directory structure

`blueprint` - this is a git submodule containing the required version of Blueprint

`config_system` - contains the Python-based configuration system

`docs` - project documentation

`example` - example files for project setup

`scripts` - miscellaneous scripts

`tests` - contains build tests

#### Go code directories

`cmd` - contains the Go code for command line tools

`core` - contains the core Go code

`graph` - contains a simple Go graph implementation

`utils` - miscellaneous utility functions in Go

### Developing for Bob

To load Bob code in a Go-aware IDE, create a workspace directory
outside the Bob tree and run:

```bash
apt-get install bindfs
export GOPATH=<workspace>
bob/scripts/setup_workspace_for_bob.bash
```

## Documentation

Detailed [documentation](docs/index.md) is in the docs directory of
the repository.