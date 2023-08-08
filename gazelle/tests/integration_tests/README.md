# Gazelle Integration Tests

This subdirectory contains integration tests that generate bazel from bob, and ensure the resulting bazel is correct
by invoking bazel build on the generated build files.

## Steps to run

```sh
cd tests/integration_tests/
bazel run @gazelle//:gazelle
bazel build //...

```
