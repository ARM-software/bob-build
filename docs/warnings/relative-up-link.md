`relative-up-link` warning
==================

## Warns when target name contains up-level references `'..'`

## Problematic code:
```bp
bob_generate_source {
    name: "my_generate",
    srcs: [
        "../subdir1/file1.cpp",
        "../subdir2/file2.cpp",
    ],
}
```

## Correct code:
```bp
bob_filegroup {
    name: "my_filegroup",
    srcs: [
        "subdir1/file1.cpp",
        "subdir2/file2.cpp",
    ],
}

bob_generate_source {
    name: "my_generate",
    srcs: [
        "main.cpp",
    ],
    filegroup_srcs: ["my_filegroup"],
}
```

## Rationale:
Use of up-level references (`..`) breaks the concept of hermeticity.
Sources for a module should be relative to its current directory or
its subdirectories.
If files outside current build file have to be used, use `bob_filegroup`
as a dependency.
