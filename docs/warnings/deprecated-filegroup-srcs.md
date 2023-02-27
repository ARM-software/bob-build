# `deprecated-filegroup-srcs` warning

## Warns when deprecated `filegroup_srcs` attribute is used

## Problematic code:

```bp
bob_binary {
    name: "test_filegroup_simple_deprecated",
    srcs: [],
    filegroup_srcs: ["forward_filegroup", "glob_main"],
}
```

## Correct code:

```bp
bob_binary {
    name: "test_filegroup_simple_deprecated",
    srcs: [
        ":forward_filegroup",
        ":glob_main",
    ],
}
```

## Rationale:

`filegroup_srcs` has been depracted in favour of the simpler target syntax in `srcs`.

This aligns with Bazel syntax which can mix filegroup targets and files in the `srcs` attribute.
