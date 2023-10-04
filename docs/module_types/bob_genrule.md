# `bob_genrule`

```bp
bob_genrule {
    name, srcs, out, tool_files, cmd,
}
```

This target generates files via a custom shell command. This is usually source
code (headers or C files), but it could be anything. A single module will
generate multiple outputs from common inputs, and the command is run exactly
once.

The command will be run once - with `${in}` being the paths in
`srcs` and `${out}` being the paths in `out`.
The source and tool paths should be relative to the directory of the
`build.bp` containing the `bob_genrule`.

Supports:

- [Gazelle generation](../../gazelle/README.md)

## Properties

|                                                    |                                                                                                                                                                                                                                                                                                                                     |
| -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`name`](properties/common_properties.md#name)     | String; required                                                                                                                                                                                                                                                                                                                    |
| [`srcs`](properties/strict_properties.md)          | List of sources; default is `[]`<br>Supports glob patterns.                                                                                                                                                                                                                                                                         |
| `out`                                              | List of strings; required<br>Names of the output files that will be generated. <br>If you want to depend upon files that are generated, they must be listed as an output, there is no implicit output/input in the new genrule. All files are treated as if they're sandboxed so the build system must know the exact files output. |
| `depfile`                                          | Boolean; default is `false`<br>Enable reading a file containing dependencies in gcc format after the command completes                                                                                                                                                                                                              |
| `tool_files`                                       | List of strings; default is `[]`<br>Local file that is used as the tool in `${location}` of the cmd.                                                                                                                                                                                                                                |
| [`tools`](./properties/strict_properties.md#tools) | List of targets; <br>Name of the module (if any) that produces the tool executable.                                                                                                                                                                                                                                                 |
| [`cmd`](./properties/strict_properties.md#cmd)     | String; required<br> The command that is to be run for this module.                                                                                                                                                                                                                                                                 |

## Example

```bp
bob_genrule {
    name: "example",
    srcs: [
        ":foo", // Depends on a module
        "before_generate.in", // Depends on a filepath
    ],
    out: ["deps.cpp"],
    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --out ${out} --expect-in before_generate.in single.cpp",
}
```
