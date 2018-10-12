# Bob Build System

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

TBD

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

### Build file format

The build files are very simple. There are no conditional or control flow statements.

#### Modules

A module starts with a module type followed by a set of properties in
`name: value` format.

```
bob_binary {
    name: "less",
    srcs: ["src/less.c"],
    ldlibs: ["-lncurses"],
}
```

Every module has a `name` property that is unique.

#### Globs

Properties taking filenames also accept glob patterns. `*` is the normal
Unix wildcard, so `src/*.c` would select all C files in the src
directory. `**` will match zero or more path elements, so `src/**.c`
will match all C files in the src directory and its subdirectories.

#### Variables

Build files may contain top-level variable assignments:

```
less_srcs = ["src/less.c"]

bob_binary {
    name: "less",
    srcs: less_srcs,
    ldlibs: ["-lncurses"],
}
```

Before they are referenced, variables can be appended to with
`+=`. After they have been referenced by a module they are immutable.

#### Comments

Comments use C-style multiline `/* */` and C++ style single-line `//`
formats.

#### Types

The type of a variable is determined by the assignment. The type of a
property is determined by the module type.

* Bool (`true` or `false`)
* Integers
* Strings (`"string"`)
* Lists of strings(`["string1", "string2"]`)
* Maps (`{ key1: "value1", key2: 10, key3: ["value3"] }`)

Maps may contain values of any type. Lists and maps may have trailing
commas after the last value.

#### Operators

Strings, lists  of strings,  and maps  can be  appended using  the `+`
operator.

#### Defaults

A defaults module can be used to repeat the same properties in
multiple modules (currently restricted to the types `bob_binary`,
`bob_static_library`, `bob_shared_library` and `bob_kernel_module`)

```
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

Defaults can name other defaults.

#### Features

In most module types, every boolean configuration item (feature) is a
map property (lower cased). Within this map the standard module
properties can be used. This appends the properties to the module when
the feature is enabled.

```
bob_binary {
    name: "less",
    srcs: ["src/less.c"],
    ldlibs: ["-lncurses"],
    use_locales: {
        srcs: ["src/locales.c"],
    }
}
```

Features cannot nest. Instead of nesting, separate features should be
used.

#### Referencing configuration values

Configuration values can be used within string properties via Go
templates. As with features, the configuration identifier is lower
cased.

```
bob_binary {
    name: "less",
    srcs: ["src/less.c"],
    ldlibs: ["-lncurses"],
    use_locales: {
        srcs: ["src/locales.c"],
        cflags: ["-DDEFAULT_LOCALE={{.default_locale}}"],
    }
}
```

#### Formatter

Bob includes a canonical formatter for blueprint files. To format a
build file in place:

```
bpfmt -w build.bp
```

The canonical format uses 4 space indent, newlines after every element
of a multi-element list, and always includes trailing commas.

## Development

### Directory structure

`blueprint` - this is a git submodule containing the required version of Blueprint

`config_system` - contains the Python-based configuration system

`scripts` - miscellaneous scripts

`tests` - contains build tests


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

TBD