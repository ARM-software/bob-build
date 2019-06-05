Kernel Modules
==============

Bob supports building out-of-tree kernel modules. This invokes the
kernel build system to do the actual build.

Normally out-of-tree kernel builds leave build output in the source
tree. Bob uses some helper scripts that avoid this.

```
bob_kernel_module {
    name: "serial_driver"
    srcs: [
        "Kbuild",
        "serial_driver.c",
        "serial_driver.h",
    ],

    kernel_dir: "/path/to/kernel/src",
    kernel_arch: "arm",
    kernel_cc: "clang",
    kernel_hostcc: "clang",
    kernel_cross_compile: "arm-linux-gnueabihf-",
    kernel_clang_triple: "arm-linux-gnueabi",

    kbuild_options: ["CONFIG_DRIVER_OPTION=y"],
}
```

The most important option here is `kernel_dir`. This needs to point to
a configured checkout of the linux kernel, as an absolute path (this
is not expected to be part of your project).

The other `kernel_*` options set corresponding kernel make variables.

|Property|Make variable|Description|
|---|---|---|
|kernel_arch|ARCH|Target architecture|
|kernel_cc|CC|C compiler for target|
|kernel_hostcc|HOST_CC|C compiler for host|
|kernel_cross_compile|CROSS_COMPILE|Toolchain prefix for GNU target tools|
|kernel_clang_triple|CLANG_TRIPLE|Target triple for clang|

`kbuild_options` can be used to set module specific CONFIG
options. These are expected to be in the out-of-tree module's `Kconfig`
file.

`make_args` can also be used to pass arbitrary make arguments to the
kernel build.

## Using symbols from another out-of-tree module

Kernel modules can use symbols from in-tree kernel modules
automatically. The in-tree module just needs to be enabled.

If a kernel module needs to use the symbols from another kernel module
in Bob, use the `extra_symbols` property and pass it the Bob module
name. The following example uses this to allow `usb_driver` to refer
to functions defined in `serial_driver` from the previous example.

```
bob_kernel_module {
    name: "usb_driver"
    srcs: [
        "Kbuild",
        "usb_driver.c",
        "usb_driver.h",
    ],

    kernel_dir: "/path/to/kernel/src",
    kernel_arch: "arm",
    kernel_cc: "clang",
    kernel_hostcc: "clang",
    kernel_cross_compile: "arm-linux-gnueabihf-",
    kernel_clang_triple: "arm-linux-gnueabi",

    extra_symbols: ["serial_driver"],
}
```
