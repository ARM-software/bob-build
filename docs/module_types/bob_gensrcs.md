# Module: bob_gensrcs

This target generates files via a custom shell command. This is usually source
code (headers or C files), but it could be anything. A single module will
generate exactly one output from common inputs, and the command is run once
per source file.

This is a new rule that is closer aligned to Android's gensrcs.

The command will be run for every source, i.e. `$(in)` being the path
of input file where `$out` being the path of input with replaced extension
`output_extension` (if any).
The source and `tool_files` paths should be relative to the directory of the
`build.bp` containing the `bob_gensrcs`.

## Full specification of `bob_gensrcs` properties

### **bob_gensrcs.name** (required)

The unique identifier that can be used to refer to this module.

All names must be unique for the whole of the build system.

### **bob_gensrcs.srcs** (required)

The list of input files or 'source' files. Wildcards can be used, although they are suboptimal;
each directory in which a wildcard is used will have to be rescanned at every
build.

Source files are relative to the directory of the `build.bp` file.

If you want to depend on the outputs of a module instead of a filepath,
you may choose to do this by adding the module.name of the module you wish to
depend upon as a src with a `:` added prefix.

#### Example

```bp
bob_gensrcs {
    name: "gensrcs_dependend",
    srcs: [
        ":generate_source_single", // Depends on a module
        "before_generate.in", // Depends on a filepath
    ],
    output_extension: "cpp",

    tool_files: ["generator.py"],
    cmd: "python $(location) --in $(in) --out $(out)",
}
```

### **bob_gensrcs.output_extension** (optional)

Extension that will be substituted for each output file.

### **bob_gensrcs.depfile** (optional)

Enable reading a file containing dependencies in gcc format after the command completes

### **bob_gensrcs.tool_files** (required unless tools is set)

Local file that is used as the tool in `$(location)` of the cmd.

### **bob_gensrcs.tools** (required unless tool_files is set)

Name of the module (if any) that produces the tool executable. Leave empty
for prebuilts or scripts that do not need a module to build them.

If the module you are depending on has variants, to depend upon a specific variant
you may affix the variant with `:<variant_name>`.

####Example

```bp
bob_gensrcs {
    name: "use_target_specific_library",
    srcs: ["libout.in"],
    tools: ["host_and_target_supported_binary_new:host"],
    output_extension: "a",
    cmd: "$(location) $(in) $(out)",
}
```

### **bob_gensrcs.cmd** (required)

The command that is to be run for this module. `bob_gensrcs` supports
various substitutions in the command, by using `${name_of_var}`. The
available substitutions are:

- `$(location)`: the path to the first entry in tools or tool_files.
- `$(location <label>)`: the path to the tool, tool_file, or src with name `<label>`. Use `$(location)` if `<label>` refers to a rule that outputs exactly one file.
- `$(in)`: one or more input files.
- `$(out)`: a single output file.
- `$(depfile)`: a file to which dependencies will be written, if the depfile property is set to true.
- `$(genDir)`: the sandbox directory for this tool; contains `$(out)`.
- `$$`: a literal $

## Transition from `bob_transform_sources` to `bob_gensrcs` guide

### Implicits

For reasons involving sandboxing and remote execution we no longer allow
implicits in inputs or outputs inside of the new rule. You may fix this by removing
the implicits and listing them explicitly in the `srcs` or `out` list respectively.

This includes `module_dir` and `src_dir` as they allow you to break sandboxing.

### Cmd

The available substitutions is lower and some names have changed. You must remove functionality
inside of your old rule that relies on substitutions no longer available.

### Examples

Below are some examples pairing old rules with their new counterpart.

```bp
bob_transform_source {
    name: "transform_source_single_dir",
    srcs: [
        "f.in",
    ],
    out: {
        match: "(.+)\\.in",
        replace: [
            // inside extra directory
            "single/$1.cpp",
            "single/$1.h",
        ],
    },
    export_gen_include_dirs: ["single/transform_source"],
    tools: ["generator.py"],
    cmd: "python ${tool} --in ${in} --gen ${out} --gen-implicit-header",
}

// As `bob_gensrcs` can output exactly one file for input
// above rule has to be split to two separate ones (.cpp and .h)

bob_gensrcs {
    name: "gensrcs_single_cpp",
    srcs: [
        "single/f1.in",
    ],
    output_extension: "cpp",
    export_include_dirs: [
        "single",
    ],
    tool_files: ["generator.py"],
    cmd: "python $(location) --in $(in) --gen $(out)",
}

bob_gensrcs {
    name: "gensrcs_single_h",
    srcs: [
        "single/f1.in",
    ],
    output_extension: "h",
    export_include_dirs: [
        "single",
    ],
    tool_files: ["generator.py"],
    cmd: "python $(location) --in $(in) --gen $(out)",
}
```
