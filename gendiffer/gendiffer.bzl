load("@io_bazel_rules_go//go:def.bzl", "go_test")

def bob_generation_test(name, bob_binary, test_data, size = "small", **kwargs):
    [
        go_test(
            name = name + "_" + backend,
            srcs = [Label(":gendiffer.go")],
            deps = [
                "@io_bazel_rules_go//go/tools/bazel:go_default_library",
            ],
            args = [
                "-backend=%s" % backend,
                "-bob_binary_path=$(rootpath %s)" % bob_binary,
                "-config_file=$(rootpath %s)" % Label(":bob.%s.config") % backend,
                "-config_json=$(rootpath %s)" % Label(":bob.%s.config.json") % backend,
            ],
            size = size,
            data = test_data + [
                bob_binary,
                Label(":bob.linux.config"),
                Label(":bob.linux.config.json"),
                Label(":bob.android.config"),
                Label(":bob.android.config.json"),
                Label(":bob.android.config.d"),
            ],
            **kwargs
        )
        for backend in ["android", "linux"]
    ]
