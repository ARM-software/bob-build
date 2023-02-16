# `generate-rule` warning

## Warns when deprecated `bob_generate_source` module is used

## Problematic code:

```bp
bob_generate_source {
    name: "my_generate",
    srcs: [
        "input.txt",
    ],
    out: ["main.cpp"],
    tools: ["tool.py"],
    cmd: "python ${tool} --in ${in} --out ${out},
}
```

## Correct code:

```bp
bob_genrule {
    name: "generate_source_single_new",
    srcs: [
        "input.txt",
    ],
    out: ["main.cpp"],
    tool_files: ["tool.py"],
    cmd: "python $(location) --in $(in) --out $(out),
}
```

## Rationale:

`bob_generate_source` contains some functionality which cannot be moved
straightforward to Android's native rules, which in turn makes the
transition to other build systems (e.g. `Bazel`) impossible.

`bob_generate_source` module is considered as **deprecated** and
`bob_genrule` should be preferred where possible.
