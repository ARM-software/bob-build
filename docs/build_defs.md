Build file format
=================

The build files are very simple. There are no conditional or control
flow statements.

## Modules

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

## Bob module types

Bob has a number of module types. The properties available in each
module type are documented in the following pages.

- [bob_alias](module_types/bob_alias.md)
- [bob_binary](module_types/bob_binary.md)
- [bob_defaults](module_types/bob_defaults.md)
- [bob_generate_shared_library](module_types/bob_generate_library.md)
- [bob_generate_source](module_types/bob_generate_source.md)
- [bob_generate_static_library](module_types/bob_generate_library.md)
- [bob_install_group](module_types/bob_install_group.md)
- [bob_kernel_module](module_types/bob_kernel_module.md)
- [bob_resource](module_types/bob_resource.md)
- [bob_shared_library](module_types/bob_shared_library.md)
- [bob_static_library](module_types/bob_static_library.md)
- [bob_transform_source](module_types/bob_transform_source.md)

## Globs

Properties taking filenames also accept glob patterns. `*` is the normal
Unix wildcard, so `src/*.c` would select all C files in the src
directory. `**` will match zero or more path elements, so `src/**.c`
will match all C files in the src directory and its subdirectories.

## Variables

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

## Comments

Comments use C-style multiline `/* */` and C++ style single-line `//`
formats.

## Types

The type of a variable is determined by the assignment. The type of a
property is determined by the module type.

* Bool (`true` or `false`)
* Integers
* Strings (`"string"`)
* Lists of strings(`["string1", "string2"]`)
* Maps (`{ key1: "value1", key2: 10, key3: ["value3"] }`)

Maps may contain values of any type. Lists and maps may have trailing
commas after the last value.

## Operators

Strings, lists  of strings,  and maps  can be  appended using  the `+`
operator.

## Defaults

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

## Features

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

## Referencing configuration values

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

## Formatter

Bob includes a canonical formatter for blueprint files. To format a
build file in place:

```
bpfmt -w build.bp
```

The canonical format uses 4 space indent, newlines after every element
of a multi-element list, and always includes trailing commas.
