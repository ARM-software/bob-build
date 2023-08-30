# `deprecated-owner-prop` warning

## Warns when a module sets `owner`

## Problematic code:

```bp
bob_binary {
    name: "bin_tagable_defaults",
    srcs: ["src.cpp"],
    defaults: ["tagable_defaults"],
    owner: "baz",
}

```

## Correct code:

```bp
bob_binary {
    name: "bin_tagable_defaults",
    srcs: ["src.cpp"],
    defaults: ["tagable_defaults"],
    tags: ["owner:baz"],
}

```

## Rationale:

We are trying to remove Bob attributes which do not have a sensible translation to Bazel. In this instance the generic tagging system in Bazel provides a sensible alternative.
