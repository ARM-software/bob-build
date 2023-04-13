# Tests: Gendiffer

These set of files enable a way within Bazel to create generation tests for bob based on file structure alone.
Given a `build.bp` and `bplist` it will track the generation of `build.ninja` or `Android.bp` changes with changes within Bob.

The main use case of this is to enable targetted testing while modernising certain test targets & also to allow easier testing of Android backend generation
without a AOSP checkout at hand.

_NOTE_: 'Diff' is required on `PATH` for gendiffer to work.

## Example

An example is setup under `tests/gendiffer/example`. It will require a dir tree structure of:

```
├── WORKSPACE
├── app
│   ├── build.bp
│   ├── bplist
│   └── plugins
│       └── Android.bp.in
└── out
    ├── linux
    │   ├── build.ninja.out
    │   ├── expectedStdout.txt
    │   └── expectedStderr.txt
    └── android
        ├── Android.bp.out
        ├── expectedStdout.txt
        └── expectedStderr.txt
```

Anything inside of the `out` folder can be automatically generated for you. `WORKSPACE` is an empty file that is used as the marker
of the root of a test directory. See the `BUILD.bazel` file setup for more information.

### Expected Failures

Adding a `expectedExitCode.int` into the output folder will check that `bob` returns that exit code.

In this situation the generated build file will not be generated so there is no need to have a Ninja or Android blueprint file.

## BUILD.bazel

For Bazel to setup these tests, you must setup a `BUILD.bazel` file to invoke the `bob_generate_tests` action.

See:

```
# tests/gendiffer/BUILD.bazel


load("//gendiffer:gendiffer.bzl", "bob_generation_test")

[bob_generation_test(
    name = file[0:-len("/WORKSPACE")],
    bob_binary = "//cmd/bob:bob",
    test_data = glob(
        include = [file[0:-len("/WORKSPACE")] + "/**"],
    ),
) for file in glob(["**/WORKSPACE"])]

```

## Updating expected outputs

To update the expected outputs locally, you must run:

`UPDATE_SNAPSNOTS="true" bazel run //tests/gendiffer:<target>`

where target is e.g. `example_linux` or `example_android`
