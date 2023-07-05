# Module: bob_genrule

This target generates files via a custom shell command. This is usually source
code (headers or C files), but it could be anything. A single module will
generate multiple outputs from common inputs, and the command is run exactly
once.

This is a new rule that is closer aligned to Android's genrule.

The command will be run once - with `${in}` being the paths in
`srcs` and `${out}` being the paths in `out`.
The source and tool paths should be relative to the directory of the
`build.bp` containing the `bob_genrule`.

## Full specification of `bob_genrule` properties

### **bob_genrule.name** (required)

The unique identifier that can be used to refer to this module.

All names must be unique for the whole of the build system.

Shared library names must begin with `lib`.

### **bob_genrule.srcs** (required)

The list of input files or 'source' files. Wildcards can be used, although they are suboptimal;
each directory in which a wildcard is used will have to be rescanned at every
build.

Source files are relative to the directory of the `build.bp` file.

If you want to depend on the outputs of a module instead of a filepath,
you may choose to do this by adding the module.name of the module you wish to
depend upon as a src with a `:` added prefix.

#### Example

```bp
bob_genrule {
    name: "generate_source_single_dependend_new",
    srcs: [
        ":generate_source_single_new", // Depends on a module
        "before_generate.in", // Depends on a filepath
    ],
    out: ["deps.cpp"],

    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --out ${out} --expect-in before_generate.in single.cpp",
}
```

### **bob_genrule.out** (required)

Names of the output files that will be generated. If you want
to depend upon files that are generated, they must be listed as an output,
there is no implicit output/input in the new genrule. All files are treated as if
they're sandboxed so the build system must know the exact files output.

### **bob_genrule.depfile** (optional)

Enable reading a file containing dependencies in gcc format after the command completes

### **bob_genrule.tool_files** (required unless tools is set)

Local file that is used as the tool in `${location}` of the cmd.

### **bob_genrule.tools** (required unless tool_files is set)

Name of the module (if any) that produces the tool executable. Leave empty
for prebuilts or scripts that do not need a module to build them.

If the module you are depending on has variants, to depend upon a specific variant
you may affix the variant with `:<variant_name>`.

####Example

```bp
bob_genrule {
    name: "use_target_specific_library_new",
    out: ["libout.a"],
    tools: ["host_and_target_supported_binary_new:host"],
    cmd: "test $$(basename ${location}) = host_binary_new && cp ${location} ${out}",
}
```

### **bob_genrule.cmd** (required)

The command that is to be run for this module. bob_genrule supports various
substitutions in the command, by using `${name_of_var}`. The
available substitutions are:

- `${location}`: the path to the first entry in tools or tool_files.
- `${location <label>}`: the path to the tool, tool_file, or src with name `<label>`. Use `${location}` if `<label>` refers to a rule that outputs exactly one file.
- `${in}`: one or more input files.
- `${out}`: a single output file.
- `${depfile}`: a file to which dependencies will be written, if the depfile property is set to true.
- `${genDir}`: the sandbox directory for this tool; contains `${out}`.
- `$$`: a literal $

## Transition from `bob_generate_sources` to `bob_genrule` guide

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
bob_generate_source {
    name: "generate_source_single",
    srcs: [
        "before_generate.in",
    ],
    out: ["single.cpp"],

    tools: ["generator.py"],
    cmd: "python ${tool} --in ${in} --out ${out} --expect-in before_generate.in",
}

bob_genrule {
    name: "generate_source_single_new",
    srcs: [
        "before_generate.in",
    ],
    out: ["single.cpp"],

    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --out ${out} --expect-in before_generate.in",
}

bob_generate_source {
    name: "generate_source_multiple_out",
    srcs: [
        "before_generate.in",
    ],
    out: [
        "multiple_out.cpp",
        "multiple_out2.cpp",
    ],

    tools: ["generator.py"],
    cmd: "python ${tool} --in ${in} --out ${out} --expect-in before_generate.in",
}

bob_genrule {
    name: "generate_source_multiple_out_new",
    srcs: [
        "before_generate.in",
    ],
    out: [
        "multiple_out.cpp",
        "multiple_out2.cpp",
    ],

    tool_files: ["generator.py"],
    cmd: "python ${location} --in ${in} --out ${out} --expect-in before_generate.in",
}

bob_generate_source {
    name: "use_target_specific_library",
    out: ["libout.a"],
    generated_deps: ["host_and_target_supported_binary:host"],
    cmd: "test $$(basename ${host_and_target_supported_binary_out}) = host_binary && cp ${host_and_target_supported_binary_out} ${out}",
    build_by_default: true,
}

bob_genrule {
    name: "use_target_specific_library_new",
    out: ["libout.a"],
    tools: ["host_and_target_supported_binary_new:host"],
    cmd: "test $$(basename ${location}) = host_binary_new && cp ${location} ${out}",
}
```
