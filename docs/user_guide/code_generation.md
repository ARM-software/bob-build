Code Generation
===============

Occasionally it's preferable to use a tool to generate source code,
rather than manually write it. Bob has a few module types that support
generating code at compile time (the collection of module types that
do this are refered to as generators). In all cases generator output
is placed in an intermediate directory specific to the module.

```
bob_generate_source {
    name: "wayland_custom_protocol_code",
    srcs: ["custom_protocol.xml"],
    outs: ["code/wayland_server_custom_protocol.c"],

    args: ["code"],
    cmd: ["wayland-scanner ${args} ${in} ${out}"],
}

bob_static_library {
    name: "wayland_custom_protocol",
    generated_sources: ["wayland_custom_protocol_code"],
}
```

This calls `wayland-scanner` (which should be available in your
`PATH`) to generate `code/wayland_server_custom_protocol.c` from
`custom_protocol.xml`. The `generated_sources` property indicates that
the non-header outputs of `wayland_custom_protocol_code` should be
passed to the compiler to create the static library.

In the `cmd` and `args` properties a number of variables are available
to be used (see [reference
documentation](../module_types/common_generate_module_properties.md)).

Note that the above module does not create a dependency on
`wayland-scanner`, in the same way that dependencies are not created
on the compilers and linkers used in C compilation.

When scripts within the project are used to generate code, use the
`tool` property to add a dependency on the script.

```
bob_generate_source {
    name: "wayland_custom_protocol_code",
    srcs: ["custom_protocol.xml"],
    outs: ["code/wayland_server_custom_protocol.c"],

    tool: "gen_wayland.py",
    args: ["code"],
    cmd: ["${tool} ${args} ${in} ${out}"],
}

bob_static_library {
    name: "wayland_custom_protocol",
    generated_sources: ["wayland_custom_protocol_code"],
}
```

Similarly, if a compiled executable must be run to generate code, then
use the `host_bin` property.

```
bob_binary {
    name: "code_generator",
    srcs: ["code_generator/main.c"],
    host_supported: true,
    target_supported: false,
}

bob_generate_source {
    name: "source_code",
    srcs: ["templates/source.in"],
    outs: ["source.c"],

    host_bin: "code_generator",
    cmd: ["${host_bin} -o ${out} ${in}"],
}

bob_static_library {
    name: "libsource",
    generated_sources: ["source_code"],
}
```

When generating header files use `export_gen_include_dirs` to indicate
the directories within the output tree that will contain the
headers. These directories will be added to the include search path of
users.

Generators can output a mix of source code and headers. If only the
headers from the generator need to be used, use `generated_headers` to
pick them up. `generated_sources` picks up the headers, and compiles
the source code.

```
bob_generate_source {
    name: "wayland_custom_client_interface",
    srcs: ["custom_protocol.xml"],
    outs: ["inc/wayland_server_custom_protocol.h"],

    export_gen_include_dirs: ["inc"],

    args: ["client"],
    cmd: ["wayland-scanner ${args} ${in} ${out}"],
}

bob_static_library {
    name: "wayland_custom_client",
    srcs: ["wayland_client.c"],
    generated_headers: ["wayland_custom_client_interface"],
}
```

## Implicit inputs and outputs

Normally a command will mention one input file, and produce one output
file. If in doubt, stick to this pattern.

Bob needs to know about all input and output files so that
dependencies can be appropriately set up. This allows incremental
builds to work correctly.

Commands that have multiple inputs/outputs are expected to be able to
identify the inputs and outputs on the command line. i.e. for multiple
inputs a command might look like `${tool} -o ${out} ${in}` where
`${in}` ends up being more than one file; similarly for multiple
outputs `${tool} -i ${in} ${out}`; for multiple inputs _and_ multiple
outputs `${tool} -i ${in} -o ${out}` can work, as long as the tool is
taught to parse the command line appropriately.

Commands that use specific flags to identify a particular output are
not well catered for. For inputs, the template `{{match_srcs
\"file.txt\"}}` can be used to name a particular file from the
module's `src` list.

The `${in}` variable contains all source files from `srcs` and
`generated_sources`. The `${out}` variable contains all the outputs
from `outs`. These are known as explicit inputs and explicit outputs
respectively, as the files are explicitly mentioned on the command
line.

If you need to name a source file that you don't want to see in
`${in}`, use `implicit_srcs`. If you need to identify an output file
that you don't want to see in `${out}`, use `implicit_outs`. These
files that do not appear on the command line are known as implicit
inputs and implicit outputs.

## Many input files with common command

`bob_generate_source` runs the given command once to produce all the
outputs from the named inputs. In the situation where you have many
input files and need to run the same command on each input file to
generate a corresponding output, use the `bob_transform_source`
module.

```
bob_transform_source {
    name: "secret_sauce",
    srcs: [
        "src/a.cpp.source",
        "src/b.cpp.source",
        "inc/a.hpp.source",
        "inc/b.hpp.source",
    ],
    outs: {
        match: "(.*)\.source",
        replace: "$1",
    },

    export_gen_include_dirs: ["inc"],

    tool: "obfuscate.py",
    cmd: "${tool} -o ${out} ${in}",
}

bob_static_library {
    name: "libsecret",
    generated_sources: ["secret_sauce"],
}
```

This will run the following 4 commands. The output filenames are
derived from the input filenames according to the regular expression
replacement defined by the `match` and `replace` properties. The
output is always placed in module specific intermediate directories.

```sh
obfuscate.py -o ${gen_dir}/src/a.cpp ${src_dir}/src/a.cpp.source
obfuscate.py -o ${gen_dir}/src/b.cpp ${src_dir}/src/b.cpp.source
obfuscate.py -o ${gen_dir}/inc/a.hpp ${src_dir}/inc/a.hpp.source
obfuscate.py -o ${gen_dir}/inc/b.hpp ${src_dir}/inc/b.hpp.source
```

## Generating binaries

To create binaries and libraries from arbitrary commands, the module
types `bob_generate_binary`, `bob_generate_static_library` and
`bob_generate_shared_library` should be used. These can be referenced
in the same places where the equivalent `bob_binary`,
`bob_static_library` and `bob_shared_library` module types are used.

One use for these module types is to extract a library from a
compressed archive with `tar` or a similar command.

```
bob_generated_shared_library {
    name: "libfoo",
    srcs: "libfoo.tar.bz2",
    headers: ["include/libfoo.h"],

    export_gen_include_dirs: ["include"],

    args: [
        "--strip-components=1",
        "-C",
        "${gen_dir}",
        "-xj",
    ],
    cmd: "tar ${args} -f ${in}",
}
```

Note that with binary generators, the name of the generated output
file is derived from the name of the module. The derived output
filename is treated as an explicit output and is available as
`${out}`. These generators may also create header files to go with the
library; these should be listed with the `headers` property. The
headers are implicit outputs and will not appear in `${out}`.

## Discovered dependencies

The dependencies of a given command may change when the input file is
changed. This happens with C source files and their header
dependencies. Bob's generators support discovering these dependencies
by using Makefile fragments that list the dependencies. The fragment
needs to be produced by the command that is generating the output, and
the `depfile` property lets Bob know to expect it.

The format of the Makefile is as follows. The dependencies can be
listed on a single line (just remove all line continuations `\`).

```Makefile
outputfile: \
 dependency1 \
 dependency2 \
 dependency3 \
 ...
 dependencyn
```

```
bob_generate_source {
    name: "wayland_custom_protocol_code",
    srcs: ["custom_protocol.xml]"
    outs: ["code/wayland_server_custom_protocol.c"],
    depfile: "out.d",

    tool: "gen_wayland.py",
    args: [
        "code",
        "-d",
        "${depfile}",
    ],
    cmd: ["${tool} ${args} ${in} ${out}"],
}

bob_static_library {
    name: "wayland_custom_protocol",
    generated_sources: ["wayland_custom_protocol_code"],
}
```

For `bob_transform_source` the dependency file name is specified using
the same regular expression as the output file.

```
bob_transform_source {
    name: "secret_sauce",
    srcs: [
        "src/a.cpp.source",
        "src/b.cpp.source",
        "inc/a.hpp.source",
        "inc/b.hpp.source",
    ],
    outs: {
        match: "(.*)\.source",
        replace: "$1",
        depfile: "$1.d",
    },
    export_gen_include_dirs: ["inc"],

    tool: "obfuscate.py",
    cmd: "${tool} -d ${depfile} -o ${out} ${in}",
}

bob_static_library {
    name: "libsecret",
    generated_sources: ["secret_sauce"],
}
```

## Chained generators

The output of a generator can be used by another generator. There are
a few different ways to specify this depending on what needs to be used
from the output.

Use `module_srcs` when every output of a generator should be used as
an input in the next generator.

Use `module_deps` when you only want to use certain outputs from the
earlier generator. The command(s) have access to the output directory
of the earlier generator.

Use `encapsulates` to make one generator wrap all the output of a set
of other generators.

The following example uses `module_srcs` to pass a generated source
file through a `bob_transform_source` as well.

```
bob_generate_source {
    name: "wayland_custom_protocol_code",
    srcs: ["custom_protocol.xml"],
    outs: ["code/wayland_server_custom_protocol.c"],

    args: ["code"],
    cmd: ["wayland-scanner ${args} ${in} ${out}"],
}

bob_transform_source {
    name: "secret_sauce",
    srcs: [
        "src/a.cpp.source",
        "src/b.cpp.source",
        "inc/a.hpp.source",
        "inc/b.hpp.source",
    ],
    module_srcs: ["wayland_custom_protocol_code"],
    outs: {
        match: "(.*)\.source",
        replace: "$1",
        depfile: "$1.d",
    },
    export_gen_include_dirs: ["inc"],

    tool: "obfuscate.py",
    cmd: "${tool} -d ${depfile} -o ${out} ${in}",
}

bob_static_library {
    name: "wayland_custom_protocol",
    generated_sources: ["secret_sauce"],
}
```

The next example shows how `module_deps` might be used. Note that for
each module named in `module_deps` there are variables to get the
intermediate directory as well as all the outputs for that module. The
variables are `${mod_dir}` and `${mod_outs}` where `mod` is the module
name.

```
bob_generate_source {
    name: "templates",
    srcs: ["module.tar.bz2"],
    outs: [
        "x.in",
        "y.in",
        "z.in",
    ],

    args: [
        "--strip-components=1",
        "-C",
        "${gen_dir}",
        "-xj",
    ],
    cmd: "tar ${args} -f ${in}",
}

bob_generate_source {
    name: "x_code",
    module_deps: ["templates"],
    outs: [
        "src/x.cpp",
        "inc/x.h",
    ],

    tool: "x_generator.py",
    cmd: "${tool} -c ${gen_dir}/src/x.cpp -h ${gen_dir}/src/x.h ${templates_dir}/x.in",
}

bob_generate_source {
    name: "y_code",
    module_deps: ["templates"],
    outs: [
        "src/y.cpp",
        "inc/y.h",
    ],

    tool: "y_generator.py",
    cmd: "${tool} -c ${gen_dir}/src/y.cpp -h ${gen_dir}/src/y.h ${templates_dir}/y.in",
}

bob_static_library {
    name: "libstuff",
    generated_sources: [
        "x_code",
        "y_code",
    ],
}
```

Here's an example of encapsulating headers generated by LLVM's tblgen
in different modes.

```
bob_generate_source {
    name: "backend_emitter",
    srcs: ["src/backend.td"],
    out: ["inc/emitter.hpp"],

    export_gen_include_dir: ["inc"],

    args: [
        "-gen-emitter",
        "-d",
        "${depfile}",
    ],
    cmd: "tblgen ${args} -o ${out} ${in}",
}

bob_generate_source {
    name: "backend_registers",
    srcs: ["src/backend.td"],
    out: ["inc/registers.hpp"],

    export_gen_include_dir: ["inc"],

    args: [
        "-gen-register-info",
        "-d",
        "${depfile}",
    ],
    cmd: "tblgen ${args} -o ${out} ${in}",
}

bob_generate_source {
    name: "backend_instructions",
    srcs: ["src/backend.td"],
    out: ["inc/intructions.hpp"],
    depfile: "deps.d",

    export_gen_include_dir: ["inc"],

    args: [
        "-gen-instr-info",
        "-d",
        "${depfile}",
    ],
    cmd: "tblgen ${args} -o ${out} ${in}",
}

bob_generate_source {
    name: "backend_interface",
    srcs: ["src/backend.td"],
    out: ["inc/disassembler.hpp"],
    depfile: "deps.d",

    export_gen_include_dir: ["inc"],

    args: [
        "-gen-disassembler",
        "-d",
        "${depfile}",
    ],
    cmd: "tblgen ${args} -o ${out} ${in}",

    // Pure encapsulation (no cmd or output file) is not currently supported.
    // Make this module encapsulate the others.
    encapsulates: [
        "backend_emitter",
        "backend_registers",
        "backend_instructions",
    ],
}

bob_shared_library {
    name: "libbackend",
    srcs: ["src/backend.cpp"],
    generated_headers: ["backend_interface"],
}
```

### Referencing C flags

A generator's command may need to use the same compiler flags that
have been used to compile a library. An example of this is if the
generator were to invoke another build system, and wanted to pass
through the same `CFLAGS`. The `flag_defaults` property can be used to
reference a `bob_default` module where the common flags are specified.

```
bob_defaults {
    name: "build_flags",
    asflags: ["-g"],
    cflags: [
        "-g",
        "-DDEBUG=1",
    ],
    cxxflags: ["--fno-rtti"],
}

bob_generate_library {
    name: "libz",
    srcs: [
        "*.c",
        "*.cpp",
        "Makefile",
    ],
    flag_defaults: ["build_flags"],
    target: "host",

    args: [
        "-C",
        "${gen_dir}",
        "ASFLAGS=${asflags}",
        "CFLAGS=${cflags}",
        "CXXFLAGS=${cxxflags}",
    ],
    cmd: "cmake ${module_dir} --build ${gen_dir} ; make ${args}",
}
```

Note that the above is not a recommendation for using Bob to invoke
other build systems. This should be avoided, as the best way to ensure
consistent builds is for the build system to be aware of the full
dependency tree.
