Features
========

The feature system allows for the possibility of changing aspects of the build
depending on the configuration.

It provides a way to select flags, source files, includes and other
settings by indicating that they should only be included when a
certain configuration option is enabled. In most cases, the settings
apply cumulatively.

## Connection to config system

The idea of the feature system is that it is supposed to work
with the output of the config system. Each config option (e.g. `DEBUG`)
generates a matching feature (`debug`).

## How a feature is set and referred to

A feature added to [Mconfig](config_system.md) (e.g. `FOO`)
becomes `CONFIG_xxx=value` (e.g. `CONFIG_FOO=value`) in
`$OUT/bob.config`. We can refer to it in Bob modules as follows:

```bp
bob_module_type {
    property1: "default_value_if_feature_not_enabled",
    feature_name: {
        property1: "foo",
        property2: ["bar", "baz"],
        property3: false,
    },
}
```

Where the properties inside `feature_name` are only set if
`CONFIG_FOO` is enabled in the current configuration. Feature-specific
properties have priority over non-feature-specific properties. Note
that where the same property is specified in multiple feature blocks,
there is no priority between the blocks. If the property is
single-valued (like `enabled`) the result is undefined. If the
property is a list then all elements will be present, but the order is
undefined.

Here's a more concrete example, with a `DEBUG` option:

```bp
bob_static_library {
    name: "libFoo",
    debug: {
        cflags: ["-DUI_DEBUG"],
    },
    cflags: ["-pthread"],
    src: ["src/foo.cpp"],
}
```
So if `debug` is enabled we will have `cflags = ["-pthread", "-DUI_DEBUG"]`

## Limitations
The feature system only supports a single level of features, and no boolean
operations (so no way to say `!release` or `debug && instrumentation`). If these
are required, then a new config option should be added to calculate this.

Features must not have the same name as any Bob module property.

## Example

This example shows a choice group of mutually exclusive colors. Each
value is a [boolean](config_system.md#booleans) property - here, if
the choice is `RED`, the file `src/red_support.cpp` is compiled.

config file:
```
choice
	prompt "Favourite color"
	default BLUE
config BLUE
	bool "Blue"
config RED
	bool "Red"
config GREEN
	bool "Green"
endchoice
```

.bp file:
```bp
bob_static_library {
    name: "libColor",
    srcs: ["src/main.cpp"],
    red: {
        srcs: ["src/red_support.cpp"],
    },
    green: {
        srcs: ["src/green_support.cpp"],
    },
    blue: {
        srcs: ["src/blue_support.cpp"],
    },
}
```
