Features
========

The feature system allows for the possibility of changing aspects of the build
depending on the configuration.

It provides a way to select flags, source files, includes and others
by indicating that they should only be included when a certain
configuration option is enabled.

## Connection to config system

The idea of the feature system is that it is supposed to work
with the output of the config system. Each config option (e.g. `DEBUG`)
generates a matching feature (`debug`).

## How a feature is set and referred to
A feature added to [Mconfig](docs/mconfig.md) (e.g. `FOO`)
becomes `CONFIG_xxx=value` (e.g. `CONFIG_FOO=value`) in
`$OUT/build.config`. We can refer to it in Bob modules as follows:

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

Where the properties inside `feature_name` are only set if `CONFIG_FOO` is
enabled in the current configuration.

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

## Templated parameters
This feature allows for string replacements, using
[Go's built-in template system](https://golang.org/pkg/text/template/).

Configuration values are provided to the Go templates as data (as a
map), and can be accessed by using keys, so `{{.param}}` will be
replaced with the value of `param` from the config. If `param` is a
boolean value, `1` will be used for true and `0` for false.

A few custom functions are implemented by Bob:

`{{to_upper .param}}` - return the parameter as upper case

`{{to_lower .param}}` - return the parameter as lower case

`{{split .param sep}}` - separate the parameter into an array on each occurence of `sep`

`{{reg_match regexp .param}}` - test if the parameter matches a regular expression

`{{reg_replace regexp .param replace_re}}` - regular expression replacement on parameter

`{{match_srcs file_glob}}` - expand to matching files in the module's
                             `srcs` property (only valid in `ldflags`,
                             `cmd` and `args`)

Go templates natively support more. [Check Go template package.](https://golang.org/pkg/text/template/)

#### Example
This example shows an enum-like config option, `COLOR`, which is chosen based on
a choice group, where it can hold only the values of selected colors. The value
of `COLOR` is substituted into a compiler flag using the Go template syntax
`{{.color}}`. In the choice group, each value can also be used as a boolean
property - here, if the color is `RED`, another source file is added (see
section "Booleans").

#### Example
config file:
```
config COLOR
	string
	default "blue" if BLUE
	default "red" if RED
	default "green" if GREEN

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
    cflags: ["-pthread", "-DCOLOR={{.color}}"],
    srcs: ["src/main.cpp"],
    red: {
        srcs: ["src/red_support.cpp"],
    },
}
```
