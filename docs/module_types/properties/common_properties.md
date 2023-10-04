# Common Properties

These properties can be set on all module types within Bob.

## `name`

The unique identifier that can be used to refer to this module.

All names **must be unique** within the scope fo the build system.

Shared library names must begin with `lib`.

---

## `srcs`

The list of source files or targets which produce source files.
Wildcards can be used, although they are suboptimal;
each directory in which a wildcard is used will have to be rescanned at every
build.

Source files, given with the parameter `srcs`, are relative to the
directory of the `build.bp` file.

An appropriate compiler will be invoked for each source file based on
its file extension. Files with an unknown extension are only allowed
if referenced by [`match_srcs`](../../strings.md#match_srcs) usage within
the module, otherwise an error will be raised.

---

## `enabled`

Boolean; default is `true`.

Used to disable the generation of build rules.
When set to `false`, no build rule will be generated.

---

## `target_supported`

Boolean; default is `true`.

If true, the module will be built using the target toolchain.

---

## `host_supported`

Boolean; default is `false`.

If true, the module will be built using the host toolchain.

---

## `target`

Property map; default is `{}`.

Allows setting attributes specifically for compilation target.

Every property a module supports, except `name` and `defaults`.
These properties will only be applied to the target version of a module.
[Features](../../features.md) can also be used inside the `target` sections.

```bp
bob_binary {
    name: "hello",
    target: {
        cflags: ["-DPLATFORM_NAME=target"],
        target_toolchain_clang: {
            cflags: ["-mtune=..."],
        },
    },
}
```

---

## `host`

Property map; default is `{}`.

Allows setting attributes specifically for the host.

Every property a module supports, except `name` and `defaults`.
These properties will only be applied to the host version of a module.
[Features](../../features.md) can also be used inside the `host` sections.

```bp
bob_binary {
    name: "hello",
    host: {
        cflags: ["-DPLATFORM_NAME=host"],
    },
}
```

---

## `tags`

List of strings; default is []

This attribute allows the user to add generic string tags to a target.

Certain tags have special functionality within Bob:
| Tag value | Behaviour |
| ----------------- | ---------------------------------------------------------------------------------------------------- |
| `"owner:<owner>"` | When building on Android this will set the Soong `owner` field and mark `vendor`, `proprietary` and `soc_specific` as `true`. |

---
