workspace(name = "bob")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_skylib",
    integrity = "sha256-vCg8381SalLDIBJ5zaS8KYZS76iYsQtNsIN9xRZSdW8=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
    ],
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

http_archive(
    name = "io_bazel_rules_go",
    integrity = "sha256-M6zErg9wUC20uJPJ/B3Xqb+ZjCPn/yxFF3QdQEmpdvg=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    integrity = "sha256-dd8ojEsxyB61D1Hi4U9HY8t1SNquEmgXJHBkY3/Z6mI=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.36.0/bazel-gazelle-v0.36.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.36.0/bazel-gazelle-v0.36.0.tar.gz",
    ],
)

http_archive(
    name = "rules_python",
    sha256 = "ca77768989a7f311186a29747e3e95c936a41dffac779aff6b443db22290d913",
    strip_prefix = "rules_python-0.36.0",
    url = "https://github.com/bazelbuild/rules_python/releases/download/0.36.0/rules_python-0.36.0.tar.gz",
)

load("@rules_python//python:repositories.bzl", "py_repositories", "python_register_toolchains")

python_register_toolchains(
    name = "python3_10",
    python_version = "3.10",
    register_coverage_tool = True,
)

py_repositories()

# TODO: Fix config system import structure
# http_archive(
#     name = "rules_python_gazelle_plugin",
#     sha256 = "ca77768989a7f311186a29747e3e95c936a41dffac779aff6b443db22290d913",
#     strip_prefix = "rules_python-0.36.0/gazelle",
#     url = "https://github.com/bazelbuild/rules_python/releases/download/0.36.0/rules_python-0.36.0.tar.gz",
# )
# load("@rules_python_gazelle_plugin//:deps.bzl", _py_gazelle_deps = "gazelle_deps")

# _py_gazelle_deps()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")  # keep

http_archive(
    name = "rules_multirun",
    sha256 = "9ced12fb88f793c2f0a8c19f498485c4a95c22c91bb51fc4ec6812d41fc3331d",
    strip_prefix = "rules_multirun-0.6.0",
    url = "https://github.com/keith/rules_multirun/archive/refs/tags/0.6.0.tar.gz",
)

local_repository(
    name = "com_github_google_blueprint",
    path = "./blueprint",
)

load("//:deps.bzl", "go_dependencies")

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()

go_rules_dependencies()

# 1.18 for latest rules_go and Gazelle.
# Bob itself supports >=1.18
go_register_toolchains(version = "1.18")

# TODO: Fix config system import structure
# load("@rules_python_gazelle_plugin//:deps.bzl", _py_gazelle_deps = "gazelle_deps")
# _py_gazelle_deps()

gazelle_dependencies()

load("@rules_python//python:pip.bzl", "pip_parse")

pip_parse(
    name = "config_deps",
    requirements_lock = "//config_system:requirements_lock.txt",
)

load("@config_deps//:requirements.bzl", "install_deps")

install_deps()
