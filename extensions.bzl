load("@bazel_gazelle//:deps.bzl", "go_repository")

def _impl_blueprint(module_ctx):
    for module in module_ctx.modules:
        for tag in module.tags.from_commit:
            go_repository(
                name = "com_github_google_blueprint",
                commit = tag.commit,
                importpath = "github.com/google/blueprint",
                patch_args = ["-p{}".format(tag.patch_strip)],
                patches = tag.patches,
            )

_from_commit = tag_class(
    attrs = {
        "commit": attr.string(
            doc = """The Blueprint Go module Commit SHA.

            Downloads Blueprint package with commit""",
            mandatory = True,
        ),
        "patches": attr.label_list(
            doc = "A list of file patches to apply to the repository.",
        ),
        "patch_strip": attr.int(
            default = 0,
            doc = "The number of leading path segments to be stripped from the file name in the patches.",
        ),
    },
    doc = "Allows to install blueprint Go module from specific commit SHA.",
)

# NOTE: This extention is to workaroud the problem with Blueprint module which contains broken paths.
# The only possible way to install it is by `go_repository` rule which allows to specify exact commit SHA.
# `bazel-gazelle` contains `go_deps` module extentions but so far it does not support specifying commit SHA.
blueprint = module_extension(
    implementation = _impl_blueprint,
    tag_classes = {
        "from_commit": _from_commit,
    },
)
