# Gazelle Bob Plugin

> ⚠ This plugin is still under development, use at your own discretion. ⚠

Bob is now deprecated.

This [Gazelle][gazelle] plugin provides a migration path from the current Bob build system to [Bazel][bazel].

By leveraging the existing Bob parser, this plugin will generate Bazel files for the supported targets.

This functionality is currently under development, this document will be updated with instructions on how to register and use the plugin once ready.

### Directives

Our Gazelle extension adds new directives.

| **Directive**                                                                                                                            | **Default value** |
| ---------------------------------------------------------------------------------------------------------------------------------------- | ----------------- |
| `# gazelle:bob_ignore`                                                                                                                   | `nil`             |
| Specificies a directory relative to the bob root to ignore when parsing build.bp files. Currently only works from the root `BUILD.bazel` |                   |

[bazel]: https://bazel.build/
[gazelle]: https://github.com/bazelbuild/bazel-gazelle
