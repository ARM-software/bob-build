# `default-srcs` warning

## Warns when `srcs`/`exclude_srcs` property is used in `bob_defaults`.

## Problematic code:

```bp
bob_defaults {
    name: "my_defaults",
    cflags: ["-DDEBUG=1", "-Wall"],
    srcs: [a.cpp],
}

bob_binary {
    name: "my_binary",
    defaults: ["my_defaults"],
    srcs: ["main.cpp"],
}
```

## Correct code:

```bp
bob_defaults {
    name: "my_defaults",
    cflags: ["-DDEBUG=1"],
}

bob_filegroup {
    name: "my_filegroup",
    srcs: [a.cpp],
}

bob_binary {
    name: "my_binary",
    defaults: ["my_defaults"],
    srcs: ["main.cpp"],
    filegroup_srcs: ["my_filegroup"],
}
```

## Rationale:

Bazel build system does not support such concept of `defaults`.
Including sources through `bob_defaults` across the modules makes
things really hard to convert and is contrary to Bazel principles.

Shared sources should use a `bob_filegroup` instead.

This warning should ease removal of global include headers since
each target will have to specify its own include paths and sources.
