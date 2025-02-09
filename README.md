# ⚠ Deprecation Notice ⚠

_Bob will be deprecated_ when [Android moves to Bazel][aosp-bazel].

We recommend that new projects use [Bazel][bazel] instead.

We will be introducing stricter build rules in Bob to improve Bazel alignment.

This may cause existing builds to break.

[aosp-bazel]: https://developers.googleblog.com/2020/11/welcome-android-open-source-project.html
[bazel]: https://bazel.build/

# Bob Build System

[![CI](https://github.com/ARM-software/bob-build/workflows/CI/badge.svg)](https://github.com/ARM-software/bob-build/actions)

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

- golang (>=1.18)
- ninja-build (>=1.8)
- python3 (>=3.6)
- python3-ply

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

`internal` - contains Go packages for internal use

`plugins` - contains plugins for Soong

### Developing for Bob

To load Bob code in a Go-aware IDE, create a workspace directory
outside the Bob tree and run:

```bash
apt-get install bindfs
export GOPATH=<workspace>
bob/scripts/setup_workspace_for_bob.bash
```

## Bazel

Bob has the minimal support for building with Bazel via Gazelle.

To build Bob:

```sh
bazelisk build //...
```

To run the Go unit tests:

```sh
bazelisk test //...
```

To update build files:

```sh
bazelisk run //:gazelle
```

To update `deps.bzl`:

```sh
bazelisk run //:gazelle-update-repos
```

### Generating Coverage

Generate the LCOV files:

```sh
bazelisk coverage --instrument_test_targets --@io_bazel_rules_go//go/config:cover_format=lcov --combined_report=lcov //...
```

Generate a html report:

```sh
genhtml --output genhtml "$(bazelisk info output_path)/_coverage/_coverage_report.dat"
```

## Documentation

Detailed [documentation](docs/index.md) is in the docs directory of
the repository.
