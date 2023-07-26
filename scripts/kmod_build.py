#!/usr/bin/env python3


import argparse
import logging
import multiprocessing
import os
import subprocess
import sys
import shutil

import copy_with_deps
import kernel_config_parser

logger = logging.getLogger(__name__)

kernel_search_paths = [
    "{kdir}/arch/{arch}/include",
    "{kdir}/arch/{arch}/include/generated",
    "{kdir}/arch/{arch}/include/generated",
    "uapi",
    "{kdir}/arch/{arch}/include/uapi",
    "{kdir}/include",
    "{kdir}/include/generated/uapi",
    "{kdir}/include/uapi",
]


def parse_kbuild_options(kbuild_options):
    """Return a dictionary of kbuild_options {option:value}"""
    options = dict()

    for option in kbuild_options:
        parts = option.split("=", 1)

        if len(parts) != 2:
            logger.error("Invalid Kbuild option: '%s'", option)
        else:
            options[parts[0]] = parts[1]

    return options


def check_kbuild_option_conflicts(kdir, option, value):
    """
    Checks whether kbuild_option is already defined in the kernel config
    causing conflicts by overriding them in EXTRA_CFLAGS
    :return: Zero(0) if there is no conflict, one(1) otherwise
    """
    message = (
        "Overridden '{0}' option in EXTRA_CFLAGS. "
        "Bob was asked to set the kernel option '{0}={1}', "
        "which is already present in the kernel's config. "
        "Please disable this option in the kernel "
        "or disable out-of-tree kernel module builds in Bob to continue."
    )

    # There are few cases when Bob is allowed to override a kernel option:
    #
    # 1. If the kernel has an option present, it is set to `y` or `m` and Bob is overriding it
    #    with the same value
    # 2. If the kernel has an option present, it is set to `n` and Bob is overriding it
    #    either with `y` or `m` or `n`
    # 3. If the kernel has an option present, it is set to any value and Bob is overriding it
    #    with the same value (imply first case)

    k_option_val = kernel_config_parser.get_value(kdir, option)

    if k_option_val:
        if not (k_option_val == value) and not (
            k_option_val == "n" and value in ["y", "m"]
        ):
            logger.error(message.format(option, value))
            return 1

    return 0


def kbuild_to_cflag(option, value):
    if value in ["m", "y"]:
        cflag = str.format("-D{}=1", option)
    elif value == "n":
        cflag = str.format("-U{}", option)
    elif value.isdigit():
        cflag = str.format("-D{}={}", option, value)
    else:
        # passing values as string needs special treatment due to shell interpretation
        # -DCONFIG_A='"string value"' conforms to #define CONFIG_A "string value"
        cflag = str.format("-D{}='\"{}\"'", option, value)

    return cflag


def build_module(
    output_dir, module_ko, kdir, module_dir, make_command, make_args, extra_cflags
):
    """
    Invoke an out of tree kernel build.
    """
    # Invoke the kernel build system
    cmd = [make_command, "-C", kdir, "M=" + module_dir, "EXTRA_CFLAGS=" + extra_cflags]
    cmd.extend(make_args)

    # Sanitize the environment - we should only use build options passed in via
    # the command line.
    env = dict(os.environ)
    for var in [
        "ARCH",
        "CROSS_COMPILE",
        "CC",
        "HOSTCC",
        "CLANG_TRIPLE",
        "KBUILD_EXTRA_SYMBOLS",
        "LD",
    ]:
        env.pop(var, None)

    try:
        subprocess.check_call(cmd, env=env)
    except subprocess.CalledProcessError as e:
        logger.error("Command failed: %s", str(e.cmd))
        sys.exit(e.returncode)

    # Copy the output of the kernel build to the directory that Bob expects
    built_files = [module_ko, "Module.symvers"]
    for built_file in built_files:
        try:
            # Don't copy if already existing in desired location
            if module_dir != os.path.abspath(output_dir):
                abs_built_file = os.path.join(module_dir, built_file)
                shutil.copy(abs_built_file, output_dir)
        except (OSError, IOError) as e:
            msg = (
                "Copy file from input path: {}\n"
                "to output path: {}\n"
                "finished with error: {}"
            )
            logger.error(msg.format(abs_built_file, output_dir, e))
            sys.exit(1)


def get_tool_abspath(tool):
    """Get absolute path to tool if argument contains a path otherwise assume it's a $PATH tool
    :param tool: path to tool or $PATH prefix
    :return: Absolute path or tool
    """
    if tool and os.path.dirname(tool):
        return os.path.abspath(tool)
    return tool


def parse_source_list(sources):
    module_sources = []
    extra_symbols = []
    for source in sources:
        if os.path.basename(source) == "Module.symvers":
            extra_symbols.append(source)
        elif os.path.splitext(source)[1] == ".ko":
            # Ignore .ko files - we will detect their symbols via their
            # corresponding Module.symvers file.
            pass
        else:
            module_sources.append(source)

    return module_sources, extra_symbols


def parse_output_list(parser, outputs):
    """When this script is called from a `genrule` module, the .ko _and_
    Module.symvers file may be listed as outputs. Filter out the symvers file.
    """
    module_output = None
    module_symvers = None
    for output in outputs:
        if os.path.basename(output) == "Module.symvers":
            if module_symvers:
                parser.error(
                    "Module.symvers specified multiple times: {} and {}".format(
                        module_symvers, output
                    )
                )
            module_symvers = output
        elif os.path.splitext(output)[1] == ".ko":
            if module_output:
                parser.error(
                    ".ko output specified multiple times: {} and {}".format(
                        module_output, output
                    )
                )
            module_output = output
        else:
            parser.error(
                "Unknown output file type: {}".format(os.path.basename(output))
            )

    if not module_output:
        parser.error("No .ko output file specified")

    return module_output


def parse_args():
    logging.basicConfig(format="%(levelname)s: %(message)s", level=logging.WARNING)

    cli_description = (
        "Encapsulate an out-of-tree kernel module build, "
        "where the build does not modify the source directory"
    )
    parser = argparse.ArgumentParser(description=cli_description)
    parser.add_argument(
        "--output",
        "-o",
        required=True,
        nargs="+",
        help="Kernel module to build (including output path)",
    )
    parser.add_argument(
        "--sources",
        "-s",
        metavar="FILE",
        nargs="+",
        required=True,
        help="Kernel module source files",
    )
    parser.add_argument(
        "--depfile",
        "-d",
        metavar="DEPFILE",
        required=True,
        help="Dependency file to generate",
    )
    parser.add_argument(
        "--common-root",
        "-r",
        required=True,
        help="Common root directory that can be stripped from source paths",
    )
    parser.add_argument(
        "--module-dir", "-m", help="Module output directory in kernel build"
    )
    parser.add_argument(
        "--jobs", "-j", metavar="N", default=None, type=int, help="Allow N jobs at once"
    )
    parser.add_argument(
        "--make-command", "-M", default="make", help="Path to `make` command"
    )

    group = parser.add_argument_group("Kernel options")
    group.add_argument(
        "--kernel", "-k", metavar="KDIR", required=True, help="Kernel directory"
    )
    group.add_argument("--cc", default=None, help="Target C compiler")
    group.add_argument("--hostcc", default=None, help="Host C compiler")
    group.add_argument("--cross-compile", default=None, help="Kernel CROSS_COMPILE")
    group.add_argument("--clang-triple", default=None, help="Kernel CLANG_TRIPLE")
    group.add_argument("--ld", default=None, help="Kernel LD")
    group.add_argument(
        "--kbuild-options",
        nargs="+",
        default=[],
        help="Kernel config options to enable, that get added to EXTRA_CFLAGS too",
    )
    group.add_argument(
        "--extra-cflags", default="", help="Options to add to EXTRA_CFLAGS as a string"
    )
    group.add_argument(
        "make_args",
        nargs=argparse.REMAINDER,
        default=[],
        help="Make variables to be set",
    )

    group = parser.add_argument_group("Dependency checking")
    group.add_argument(
        "--include-dir",
        "-I",
        metavar="INCLUDE_DIR",
        action="append",
        default=[],
        help="Include file search path",
    )

    args = parser.parse_args()

    args.module_sources, args.extra_symbols = parse_source_list(args.sources)
    args.output = parse_output_list(parser, args.output)

    return args


def main():
    args = parse_args()

    # The build is run in $KDIR, rather than the usual build workdir, so
    # parameters need to be absolute so they are accessible with a different CWD.
    output_dir = os.path.dirname(args.output)
    abs_kdir = os.path.abspath(args.kernel)
    search_path = [os.path.abspath(d) for d in args.include_dir]

    cross_compile = get_tool_abspath(args.cross_compile)
    target_cc = get_tool_abspath(args.cc)
    host_cc = get_tool_abspath(args.hostcc)
    make_command = get_tool_abspath(args.make_command)

    # Check the kernel ARCH
    arch = kernel_config_parser.get_arch(abs_kdir)
    if not arch:
        sys.exit(1)

    kbuild_cflags = []
    kbuild_conflicts = 0
    # Parse kbuild_options and make them cflags style
    for option, value in parse_kbuild_options(args.kbuild_options).items():
        kbuild_conflicts += check_kbuild_option_conflicts(abs_kdir, option, value)
        kbuild_cflags.append(kbuild_to_cflag(option, value))

    if kbuild_conflicts > 0:
        sys.exit(1)

    # Prepend EXTRA_CFLAGS with modified include paths
    includes = ["-I" + s for s in search_path]
    extra_cflags = (
        " ".join(includes) + " " + args.extra_cflags + " " + " ".join(kbuild_cflags)
    )

    # If autoconf.h is older than kernel config, the kernel needs to be rebuilt
    # to update this file.
    autoconf_file = os.path.join(abs_kdir, "include", "generated", "autoconf.h")
    kernel_config_file = kernel_config_parser.get_config_file_path(abs_kdir)
    if os.path.exists(autoconf_file):
        if os.path.getmtime(autoconf_file) < os.path.getmtime(kernel_config_file):
            msg = "%s is older than %s. make modules_prepare needs to be run."
            logger.warning(msg, autoconf_file, kernel_config_file)
    else:
        msg = "%s not found. make modules_prepare needs to be run"
        logger.warning(msg, autoconf_file)

    deps = []

    # Add commonly needed search paths for copy_with_deps
    search_path.extend(
        [str.format(d, kdir=abs_kdir, arch=arch) for d in kernel_search_paths]
    )
    kconfig = os.path.join("linux", "kconfig.h")
    root = os.path.abspath(args.common_root)
    for src in args.module_sources:
        src_rel = os.path.relpath(os.path.abspath(src), root)
        if src_rel.startswith("../"):
            msg = "Source path: %s doesn't share common root directory: %s"
            logger.error(msg, src, args.common_root)
            sys.exit(1)

        dest = os.path.join(output_dir, src_rel)
        deps.extend(copy_with_deps.copy_with_deps(src, dest, search_path, [kconfig]))

    deps = sorted(set(deps))

    # Add a dependency on copy_with_deps.py, which won't have been set by Bob
    deps.append(os.path.join(os.path.dirname(sys.argv[0]), "copy_with_deps.py"))

    # Add a dependency on the test kernel Makefile. We do not attempt to add
    # dependencies on every part of the kernel's build system - this is just
    # enough to ensure that incremental builds of the Bob tests work OK.
    deps.append(os.path.join(abs_kdir, "Makefile"))

    # Add a dependency to kernel config because we are parsing it to
    # populate CFLAGS.
    deps.append(kernel_config_file)

    copy_with_deps.write_depfile(args.depfile, args.output, deps)

    make_args = args.make_args
    make_args.extend(args.kbuild_options)
    make_args.append("ARCH=" + arch)

    # CROSS_COMPILE is still required with CC=clang
    if cross_compile:
        make_args.append("CROSS_COMPILE=" + cross_compile)
    if target_cc:
        make_args.append("CC=" + target_cc)
    if host_cc:
        make_args.append("HOSTCC=" + host_cc)
    if args.clang_triple:
        make_args.append("CLANG_TRIPLE=" + args.clang_triple)
    if args.ld:
        make_args.append("LD=" + args.ld)
    elif kernel_config_parser.option_enabled(abs_kdir, "CONFIG_LD_IS_LLD"):
        # Auto-set LD to `ld.lld` if LTO has been enabled
        make_args.append("LD=ld.lld")
    if args.extra_symbols:
        extra_symbols = [os.path.abspath(d) for d in args.extra_symbols]
        make_args.append("KBUILD_EXTRA_SYMBOLS=" + " ".join(extra_symbols))

    if args.jobs:
        make_args.append("-j" + str(args.jobs))
    else:
        nproc = os.getenv("NPROC", multiprocessing.cpu_count())
        make_args.append("-j" + str(nproc))

    module_ko = os.path.basename(args.output)
    abs_module_dir = os.path.abspath(args.module_dir)
    build_module(
        output_dir,
        module_ko,
        abs_kdir,
        abs_module_dir,
        make_command,
        make_args,
        extra_cflags,
    )


if __name__ == "__main__":
    main()
