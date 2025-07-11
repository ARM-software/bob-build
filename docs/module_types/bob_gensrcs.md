# `bob_gensrcs`

```bp
bob_gensrcs {
    name, srcs, output_extension, tool_files, cmd,
}
```

This target generates files via a custom shell command. This is usually source
code (headers or C files), but it could be anything. A single module will
generate exactly one output from common inputs, and the command is run once
per source file.

This is a new rule that is closer aligned to Android's gensrcs.

The command will be run for every source, i.e. `${in}` being the path
of input file where `${out}` being the path of input with replaced extension
`output_extension` (if any).
The source and `tool_files` paths should be relative to the directory of the
`build.bp` containing the `bob_gensrcs`.

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
| `output_extension`                                 | Extension that will be substituted for each output file.                                                                                                                                                                                                                                                                            |

## Example

```bp
bob_gensrcs {
    name: "gensrcs_single_cpp",
    srcs: [
        "single/f1.in",
    ],
    output_extension: "cpp",
    export_include_dirs: [
        "single",
    ],
    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --gen ${out}",
}
```
