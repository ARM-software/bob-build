load("@io_bazel_rules_go//go:def.bzl", "go_test")

def bob_generation_test(name, bob_binary, test_data, size = "small", **kwargs):
    [
        go_test(
            name = name + "_" + backend,
            srcs = [Label("//gendiffer:gendiffer.go")],
            deps = [
                "@io_bazel_rules_go//go/tools/bazel:go_default_library",
            ],
            args = [
                "-backend=%s" % backend,
                "-bob_binary_path=$(rootpath %s)" % bob_binary,
                "-config_file=$(rootpath %s)" % Label("//gendiffer:bob.%s.config") % backend,
                "-config_json=$(rootpath %s)" % Label("//gendiffer:bob.%s.config.json") % backend,
            ],
            size = size,
            data = test_data + [
                bob_binary,
                Label("//gendiffer:bob.linux.config"),
                Label("//gendiffer:bob.linux.config.json"),
                Label("//gendiffer:bob.android.config"),
                Label("//gendiffer:bob.android.config.json"),
                Label("//gendiffer:bob.android.config.d"),
            ],
            testonly = False,
            **kwargs
        )
        for backend in ["android", "linux"]
    ]
