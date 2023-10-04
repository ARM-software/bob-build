# Migration from `bob_transform_source` to `bob_gensrcs`

## Implicits

For reasons involving sandboxing and remote execution we no longer allow
implicits in inputs or outputs inside of the new rule. You may fix this by removing
the implicits and listing them explicitly in the `srcs` or `out` list respectively.

This includes `module_dir` and `src_dir` as they allow you to break sandboxing.

## Cmd

The available substitutions is lower and some names have changed. You must remove functionality
inside of your old rule that relies on substitutions no longer available.

## Examples

Below are some examples pairing old rules with their new counterpart.

### Replacing match expressions

```bp
bob_transform_source {
    name: "before",
    srcs: [
        "f.in",
    ],
    out: {
        match: "(.+)\\.in",
        replace: [
            // inside extra directory
            "single/$1.cpp",
            "single/$1.h",
        ],
    },
    export_gen_include_dirs: ["single/transform_source"],
    tools: ["generator.py"],
    cmd: "python ${tool} --in ${in} --gen ${out} --gen-implicit-header",
}

// As `bob_gensrcs` can output exactly one file for input
// above rule has to be split to two separate ones (.cpp and .h)

bob_gensrcs {
    name: "after_a",
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

bob_gensrcs {
    name: "after_b",
    srcs: [
        "single/f1.in",
    ],
    output_extension: "h",
    export_include_dirs: [
        "single",
    ],
    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --gen ${out}",
}
```
