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
│  │ Config  ├─────►│  Context  ├───┬──►│ MconfigParser   ├───────────┐                │
│  │         │      │           │   │   │                 │           │                │
│  └─────────┘      └───────────┘   │   └─────────────────┘           │                │
│                       ▲    ▲      │                                 │                │
│                       │    │      │   ┌─────────────────┐           │                │
│                       │    │      │   │                 │           │                │
│                       │    │      └──►│ BlueprintParser ├──────┐    │                │
│                       │    │          │                 │      │    │                │
│                       │    │          └─────────────────┘      │    │                │
│                       │    │                                   │    │                │
│                       │    │     ValidateAndRegister()         │    │                │
│                       │    └───────────────────────────────────┘    │                │
│                       │                                             │                │
│                       │                                             │                │
│                       │          Register()                         │                │
│                       └─────────────────────────────────────────────┘                │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────┐
│ GenerateRules()                                                                      │
│                                                                                      │
│   ┌─────────────────┐                    ┌─────────────┐             ┌────────────┐  │
│   │                 │  MakeBuilders()    │             │   Build()   │            │  │
│   │ config[relPath] ├───────────────────►│ ruleBuiler  ├────────────►│ *rule.Rule │  │
│   │                 │                    │             │             │            │  │
│   └─────────────────┘                    ├─────────────┤             └────────────┘  │
│                                          └─────────────┘                             │
│                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

Any Gazelle plugin requires a parser and a rule set.
In the case of Bob we actually require two parsers, one for Mconfig and one for Blueprint.
Fortunately we can reuse existing rules for C/C++ since we are not adding a new language support.

In a regular plugin design it would be commonplace to parse files given as an argument to `GenerateRules`.
Unfortunately this is not easily done with Bob, as both Mconfig and Blueprint parse files recursively.
Whilst it's possible to parse a single Blueprint file at a time, in order to resolve `bob_defaults` we must
have the entire build DAG available.

For this reason the key design is to do the Parsing during the call to `Configure`. The `Context` object stores the
Bob workspace data, including the registered modules and the root path which is required to resolve Bazel targets.

### Target Mapping

In Bob, all targets are global and their names are unique across the workspace.
This means that the generator needs to translate Bob targets to Bazel targets to create the correct `srcs`, `data` and `deps` attributes.

The `Context` object stores this mapping and needs to be injected into the `ruleBuilder`.

### `bob_defaults` resolution

To support Bob features, the plugin reuses the existing Bob mutators for handling defaults:

```go
	bp.RegisterBottomUpMutator("default_deps1", bob.DefaultDepsStage1Mutator).Parallel()
	bp.RegisterBottomUpMutator("default_deps2", bob.DefaultDepsStage2Mutator).Parallel()
	bp.RegisterBottomUpMutator("default_applier", bob.DefaultApplierMutator).Parallel()
```

This simply flattens all the attributes for us before parsing.

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

We can implement a simple interfaces that accepts JSON requests over stdin:

```python
def main(stdin, stdout):
    ignore_missing = False

    for request in stdin:
        request = json.loads(request)

        repo_root = request["repo_root"]
        rel_package_path = request["rel_package_path"]

        lexer = lex_wrapper.LexWrapper(ignore_missing)
        lexer.source(Path(repo_root, rel_package_path, 'Mconfig'))

        configuration = syntax.parser.parse(None, debug=False, lexer=lexer)

        print(json.dumps(sanitize(configuration)), end="", file=stdout, flush=True)
        stdout.buffer.write(bytes([0]))
        stdout.flush()
```

This is based on the existing implementation in `config_system/config_system/data.py`.

Note however that the current parser implementation automatically crawls the directory on the `source` command:

```python
    def source(self, fname):
        """Handle the source command, ensuring we open the file relative to
        the directory containing the first Mconfig."""
        if self.root_dir is not None:
            fname = os.path.join(self.root_dir, fname)

        self.open(fname)
```

This is actually not ideal because Gazelle works on directory level. However Mconfig flags are global in scope, meaning that there is no restriction at using a flag declared in `subdir1/` for a Blueprint file in `subdir2/`.
This means that all the configs need to be parsed before rule generation takes place.

One major downside of parsing at root repo level only is that **all** of the config flags will be generated into a single BUILD.bazel file.
This would result in a massive file, ideally we should make the changes to the parser to only parse the current file and do this for every directory at Config time.

#### Interfacing with Go plugin

One of the tricky challenges is managing calls to the Mconfig parser from Go, fortunately [rules_python](https://github.com/bazelbuild/rules_python) has already solved this problem by instantiating a long running Python process for the parser (as above).
From there we simply marshal the request struct and feed it into the parser which returns the parsed configuration as a JSON object back to the plugin.

### Blueprint

The Blueprint parser is relatively straightforward, once all the factories have been registered we simply need to find all the files for Blueprint to process and fire the `ResolveDependencies` call:

```go
	bpToParse := findBpFiles(bobRootPath)
	bp.ParseFileList(bobRootPath, bpToParse, nil)
	bp.ResolveDependencies(nil)
```

This is enough for Blueprint to run all the mutators we have registered and by this point we will have a map of intermediate objects ready for processing.

## Plugin Config (Directives)

### bob_root

Marks the root directory for Bob files.
Can be used multiple times to support Monorepos.

### bob_cull_missing_deps

Optionally enable deleting rules if missing dependencies are required.
Allows for a green Bazel build during migration.
