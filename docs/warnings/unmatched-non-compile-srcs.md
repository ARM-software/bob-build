# `unmatched-non-compile-srcs` warning

## Warns when a provided non-compilable source file has not been matched by `{{match_srcs}}`

## Problematic code:

```bp
bob_generate_source {
    name: "example",
    srcs: [
        "input.c",
        "input.txt",
    ],
    out: ["out.c"],
    cmd: "foo {{match_srcs \"input.c\"}} > ${out}",
}
```

## Correct code:

```bp
bob_generate_source {
    name: "example",
    srcs: [
        "input.c",
        "input.txt",
    ],
    out: ["out.c"],
    cmd: "foo {{match_srcs \"input.txt\"}} {{match_srcs \"input.c\"}} > ${out}",
}
```

## Current issues

This warning can be triggered on modules without any `{{match_srcs}}`.

## Rationale:

Historically this check was used to ensure no unintended files were being consumed by a module.
