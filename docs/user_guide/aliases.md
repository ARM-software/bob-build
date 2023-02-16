# Building Particular Targets

As you add build objects and configurations, you may find that you
don't want certain objects to build in particular configurations, or
at least they should not be built unless specified on the command
line. The `enabled` and `build_by_default` properties allow this to be
controlled.

If not specified, all modules defined in build definitions are
enabled. That is, Bob will output a Ninja build statement for
them. This means that you can request the module is built by naming it
on the command line.

Setting the `enabled` property to `false` stops the Ninja build
statement from being generated. It will not be possible to build the
module. Generally this is done for a particular configuration, so that
Ninja never attempts to build modules that can't build in the
configuration.

```
# Mconfig
config HAVE_NCURSES
    bool "Have ncurses system library"
    default y

// build.bp
bob_binary {
    name: "less",
    srcs: ["less.c"]
    ldlibs: ["-lncurses"],

    enabled: false,
    have_ncurses: {
        enabled: true,
    },
}
```

The above says that the `less` binary links against the system library
`ncurses`. The default configuration assumes it will be available, so
`less` will normally build. However if the user sets `HAVE_NCURSES` to
`n` then we won't try to build `less`, as it would presumably fail to
link.

For the rest of this chapter we assume that the build command is setup
to be `buildme`

## Build by default

The `build_by_default` property controls whether a module is built
when the user invokes a build without specifying an explicit target.

```
# Mconfig
config DEBUG
    bool "Enable debugging"
    default n

// build.bp
bob_binary {
    // A helper binary that's useful when debugging
    name: "strace",
    srcs: ["strace.c"]

    build_by_default: false,
    debug: {
        build_by_default: true,
    },
}
```

The above definition says that the normal configuration has `DEBUG`
disabled, and `buildme` will not build `strace`. However if the user
runs `buildme strace` then `strace` will be built. If the user changes
the configuration to enable `DEBUG`, then `strace` is also built if
`buildme` is run.

This can be used to reduce build times in the common case by only
building what is usually needed. For example you may choose to exclude
extra tools or tests from the default build.

Note that the default setting for `build_by_default` is `true` for
target binaries, and `false` for everything else. This means that if
you don't specify anything, Bob tries to build all target binaries as
well as their build and install dependencies.

## Aliases

Related to these settings are aliases. These are build targets that
point to other Bob defined modules. They can be setup in two ways:

```
bob_binary {
    name: "test1",
    srcs: ["test1.c"],
}

bob_binary {
    name: "test2",
    srcs: ["test2.c"],
}

bob_binary {
    name: "test3",
    srcs: ["test3.c"],

    add_to_alias: ["tests"],
}

bob_binary {
    name: "test4",
    srcs: ["test4.c"],

    add_to_alias: ["tests"],
}

bob_alias {
    name: "tests",
    srcs: [
        "test1",
        "test2",
    ],
}
```

`test1`, `test2`, `test3`, `test4` are all in the `tests` alias. When
`buildme tests` is run, all these modules and their dependencies are
built and installed.

If the targets of the alias is a small set, then specifying them all
in the `srcs` property of the `bob_alias` is simple. This can get
difficult to maintain if there are many targets spread across several
directories. In that case, using `add_to_alias` on each module can
simplify managing them.

Note that aliases auto-ignore disabled modules. i.e. if `test1` was
disabled, then `buildme tests` would build `test2`, `test3` and
`test4`. This simplifies defining the aliases when you have a
complicated configuration.
