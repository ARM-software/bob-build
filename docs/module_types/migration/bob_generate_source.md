# Migration from `bob_generate_source` to `bob_genrule`

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

### Single in, single out

```bp
bob_generate_source {
    name: "before",
    srcs: [
        "before_generate.in",
    ],
    out: ["single.cpp"],

    tools: ["generator.py"],
    cmd: "python ${tool} --in ${in} --out ${out} --expect-in before_generate.in",
}

bob_genrule {
    name: "after",
    srcs: [
        "before_generate.in",
    ],
    out: ["single.cpp"],

    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --out ${out} --expect-in before_generate.in",
}
```

### Single in, multiple out

```bp
bob_generate_source {
    name: "before",
    srcs: [
        "before_generate.in",
    ],
    out: [
        "multiple_out.cpp",
        "multiple_out2.cpp",
    ],

    tools: ["generator.py"],
    cmd: "python ${tool} --in ${in} --out ${out} --expect-in before_generate.in",
}

bob_genrule {
    name: "after",
    srcs: [
        "before_generate.in",
    ],
    out: [
        "multiple_out.cpp",
        "multiple_out2.cpp",
    ],

    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --out ${out} --expect-in before_generate.in",
}
```

### Host binary

```bp
bob_generate_source {
    name: "before",
    out: ["libout.a"],
    generated_deps: ["host_and_target_supported_binary:host"],
    cmd: "test $$(basename ${host_and_target_supported_binary_out}) = host_binary && cp ${host_and_target_supported_binary_out} ${out}",
    build_by_default: true,
}

bob_genrule {
    name: "after",
    out: ["libout.a"],
    tools: ["host_and_target_supported_binary_new:host"],
    cmd: "test $$(basename ${location}) = host_binary_new && cp ${location} ${out}",
}
```
