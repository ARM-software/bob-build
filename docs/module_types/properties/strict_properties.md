# Common Strict Properties

These properties can only be set on strict modules.

## `srcs`

List of sources; default is `[]`

A list of source files or other targets which provide source files such as globs and filegroups.

Example:

```bp
bob_filegroup {
    name: "example",
    srcs: [
        "src.c",
        ":other_module",
    ],
}
```

## `tools`

List of targets; required unless `tool_files` is given.

Name of the module (if any) that produces the tool executable. Leave empty
for prebuilts or scripts that do not need a module to build them.

If the module you are depending on has variants, to depend upon a specific variant
you may affix the variant with `:<variant_name>`.

### Example

```bp
bob_genrule {
    name: "use_target_specific_library_new",
    out: ["libout.a"],
    tools: ["host_and_target_supported_binary_new:host"],
    cmd: "test $$(basename ${location}) = host_binary_new && cp ${location} ${out}",
}
```

## `cmd`

The command that is to be run for this module. `bob_genrule` supports various
substitutions in the command, by using `${name_of_var}`. The
available substitutions are:

- `${location}`: the path to the first entry in tools or tool_files.
- `${location <label>}`: the path to the tool, tool_file, or src with name `<label>`. Use `${location}` if `<label>` refers to a rule that outputs exactly one file.
- `${in}`: one or more input files.
- `${out}`: a single output file.
- `${depfile}`: a file to which dependencies will be written, if the depfile property is set to true.
- `${genDir}`: the sandbox directory for this tool; contains `${out}`.
- `$$`: a literal $
