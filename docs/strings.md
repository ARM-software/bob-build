String Manipulation
===================

Most strings in build definitions can use Go's built-in
[template system](https://golang.org/pkg/text/template/). Exceptions:
the `name` and `defaults` properties cannot use templates.

Go templates natively support many operations, see the documentation
in the link above.

## Configuration values

Configuration values are provided to the Go templates as data (as a
map), and can be accessed by using keys, so `{{.param}}` will be
replaced with the value of `PARAM` from the config. If `PARAM` is a
boolean value, `1` will be used for true and `0` for false.

## Custom template functions

Bob implements a few template functions. Most of these manipulate
configuration values in different ways.

In the following documentation template arguments starting with a
period (.) indicate that the argument is expected to be a
configuration value, but it can also be a string. When it is a
configuration value the function operates on the configured value.

Note that string arguments must be in double quotes, and these need to
be escaped with a backslash.

### to_upper

    {{to_upper .param}}

Return the value of `.param`, with all characters upper cased.

### to_lower

    {{to_lower .param}}

Return the value of `.param`, with all characters lower cased.

### split

    {{split .param sep}}

Separate the value of `.param` into an array on each occurrence of the
string `sep`.

### reg_match

    {{reg_match regexp .param}}

Test if the value of `.param` matches the regular expression in
`regexp`. The result is 1 (true) or 0 (false).

### reg_replace

    {{reg_replace regexp .param replace_re}}

Transform the value of `.param` as directed by `regexp` and
`replace_re`. This is a standard regular expression replace operation.

### match_srcs

    {{match_srcs file_glob}}

Return paths of files that match the glob `file_glob` from the
module's `srcs` property. This function can only be used in the
`ldflags`, `cmd` or `args` properties. Other properties are not
expected to reference files.

It is an error if no files match.

The intention of this function is to allow a command to reference a
specific file in the source tree. By looking for the file in `srcs` we
can be sure that there is a dependency on the file.

### add_if_supported

    {{add_if_supported compiler_flag}}

Return `compiler_flag` if the compiler for the module recognises it as
a valid compiler argument. Otherwise the result is the empty string.
This function can only be used in the `cflags`, `conlyflags`,
`cxxflags`, and `export_cflags` properties.

This is primarily intended to add warning flags to the build without
breaking older compilers. This should not be used to add compiler
flags that are required for functional code - as this would just move
the error from compile time to run time.

## Example

This example has a [string](config_system.md#strings) config option,
`COLOR`. The value of `COLOR` is substituted into a compiler flag
using the Go template syntax `{{.color}}`. It is separately upper
cased into a different option.

The linker script `color.lnk` is added to `ldflags` using `{{match_srcs}}`.

The warning flag `-Wno-unreachable-code-loop-increment` is added if
supported by the compiler.

config file:
```
config COLOR
	string
	default "blue"
```

.bp file:
```bp
bob_shared_library {
    name: "libColor",
    srcs: [
        "src/main.cpp",
        "color.lnk",
    ],
    cflags: [
        "-pthread",
        "-DCOLOR={{.color}}",
        "-DUPPERCASE_COLOR={{to_upper .color}}",
        "{{add_if_supported \"-Wno-unreachable-code-loop-increment\"}}",
    ],
    ldflags: [
        "-Wl,--script={{match_srcs \"color.lnk\"}}",
    ],
}
```
