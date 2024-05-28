# Overview

This document covers the design for the Bob Gazelle plugin.

The companion document is the [Gazelle design document](https://github.com/bazelbuild/bazel-gazelle/blob/master/Design.rst) which covers the overall design of Gazelle and should
shed some light on how this should work.

## Top Level Design

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                      │
│ Configure()                                                                          │
│                                                                                      │
│  ┌─────────┐      ┌───────────┐       ┌─────────────────┐                            │
│  │         │      │           │       │                 │                            │
│  │ Config  ├─────►│  Files[]  ├───┬──►│ MconfigParser   ├───────────┐                │
│  │         │      │           │   │   │                 │           │                │
│  └─────────┘      └───────────┘   │   └─────────────────┘           │                │
│                       ▲    ▲      │                                 │                │
│                       │    │      │   ┌─────────────────┐           │                │
│                       │    │      │   │                 │           │                │
│                       │    │      └──►│ BlueprintParser ├──────┐    │                │
│                       │    │          │                 │      │    │                │
│                       │    │          └─────────────────┘      │    │                │
│                       │    │                                   │    │                │
│                       │    │     Parse()                       │    │                │
│                       │    └───────────────────────────────────┘    │                │
│                       │                                             │                │
│                       │                                             │                │
│                       │          Parse()                            │                │
│                       └─────────────────────────────────────────────┘                │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘

┌───────────────────────────────────────────────────┐
│ GenerateRules()                                   │
│                                                   │
│   ┌─────────────────┐             ┌────────────┐  │
│   │                 │   Build()   │            │  │
│   │ config[relPath] ├────────────►│ *rule.Rule │  │
│   │                 │             │            │  │
│   └─────────────────┘             └────────────┘  │
│                                                   │
│                                                   │
└───────────────────────────────────────────────────┘
```

Any Gazelle plugin requires a parser and a rule set.
In the case of Bob we actually require two parsers, one for Mconfig and one for Blueprint.
Fortunately we can reuse existing rules for C/C++ since we are not adding a new language support.

In a regular plugin design it would be commonplace to parse files given as an argument to `GenerateRules`.
Unfortunately this is not easily done with Bob, as both Mconfig and Blueprint may have dependencies
on each other. For this reason the parsing is done at `Configure()` call time and the ASTs are stored
in the config struct in memory.

### Target Mapping

In Bob, all targets are global and their names are unique across the workspace.
This means that the generator needs to translate Bob targets to Bazel targets to create the correct `srcs`, `data` and `deps` attributes.

The `Mapper` object stores this mapping across a Bob build project and is shared between the Mconfig and Blueprint builders.

### Bob feature handling

The plugin needs to translate Bob features into Select statements. There are two ways to handle this.
The first way is to generate all variants of attribute values for given feature set.
For example, given `featureA` and `featureB` both set `Srcs` we would end up with the following:

```python
cc_library(
    name = "some_target",
    srcs = select({
        ":featureA": ["foo.c", "baz.c"],
        ":featureB": ["bar.c", "baz.c"],
        ":featureA_featureB": ["foo.c", "bar.c", "baz.c"]
        "//conditions:default": ["baz.c"],
    }),
)
```

Where `featureA_featureB` is a `config_setting_group`:

```python
selects.config_setting_group(
    name = "featureA_featureB",
    match_all = [
        ":featureA",
        ":featureB",
    ],
)
```

This creates quite a messy Build file however, if we can assume all features are additive (which they are not currently), we can simplify:

```python
cc_library(
    name = "some_target",
    srcs = ["baz.c"] + select({
        ":featureA": ["foo.c", ],
        "//conditions:default": [],
    }) + select({
        ":featureB": ["bar.c", ],
        "//conditions:default": [],
    }),
)
```

The advantage of this method is that we do not need to compute all possible combinations of features and it's a little easier to read.

#### `SourceProps`

As mentioned above, `SourceProps` is not purely additive when used in Features.
For example:

```
bob_static_library {
    name: "bob_test_simple_static_lib",
    srcs: ["helper_something.c"],
    some_feature: {
        exclude_srcs: ["helper_something.c"],
    },
}
```

In the above example, it's possible to exclude a source file from the parent struct conditionally. This means within the plugin we
would have to resolve all possible feature combinations and values. To simplify the plugin this will not be supported.
The above should be refactored to:

```
bob_static_library {
    name: "bob_test_simple_static_lib",
    not_some_feature: {
        srcs: ["helper_something.c"],
    },
}
```

### Bob module attribute parsing

The current approach here is to use reflection and map attributes like so

```
"Srcs": map[string]{
  "//conditions:default": ["main.c"],
  "featureA": ["foo.c"],
},
```

This map of maps is easily constructed with this approach and allows a natural generation of select statements based on the features.
This saves us code at the cost of complexity, there is no need for type assertions and multiple handlers per bob type.

## Parser Integration

### Mconfig

The current Mconfig parser is implemented in Python, the final user configuration is passed into Bob as a file.
To be able to generate all the user flags however, we need to return the raw parsed configuration.

We can implement a simple interfaces that accepts JSON requests over stdin, see `config_system/get_configs_gazelle.py`.

This is based on the existing implementation in `config_system/config_system/data.py`.

We set `ignore_source = True` to ensure that the parser is not recursive.

The parsed config is returned as JSON to the plugin.

#### Interfacing with Go plugin

One of the tricky challenges is managing calls to the Mconfig parser from Go, fortunately [rules_python](https://github.com/bazelbuild/rules_python) has already solved this problem by instantiating a long running Python process for the parser (as above).
From there we simply marshal the request struct and feed it into the parser which returns the parsed configuration as a JSON object back to the plugin.

### Blueprint

The Blueprint parser is relatively straightforward:

```go
	f, err := os.OpenFile(absolute, os.O_RDONLY, 0400)
    ast, errs := parser.ParseAndEval(filepath.Join(pkgPath, "build.bp"), f, p.scope)
```

The important caveat here is that the plugin needs to manage the scope for the `parser.File` object.
