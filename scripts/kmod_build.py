#!/usr/bin/env python

# Copyright 2018 Arm Limited.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
    "{kdir}/arch/{arch}/include/generated", "uapi",
    "{kdir}/arch/{arch}/include/uapi",
    "{kdir}/include",
    "{kdir}/include/generated/uapi",
    "{kdir}/include/uapi"
]

def kbuild_to_cflag(option):
    parts = option.split("=", 1)

    if len(parts) != 2:
        logger.error("Invalid Kbuild option: '%s'", option)

    key, value = parts[0], parts[1]

    if value in ['m', 'y']:
        cflag = str.format("-D{}=1", key)
    elif value =='n':
        cflag = str.format("-U{}", key)
    else:
        cflag = str.format("-D{}={}", key, value)

    return cflag

def build_module(output_dir, module_ko, kdir, module_dir, make_args, extra_cflags):
    """
    Invoke an out of tree kernel build.
    """
    # Invoke the kernel build system
    cmd = ["make", "-C", kdir, "M="+module_dir, "EXTRA_CFLAGS="+extra_cflags]
    cmd.extend(make_args)
    try:
        subprocess.check_call(cmd)
    except subprocess.CalledProcessError as e:
        logger.error("Command failed: %s", str(e.cmd))
        sys.exit(e.returncode)

    # Copy the output of the kernel build to the directory that Bob expects
    built_kmod = os.path.join(module_dir, module_ko)
    built_symvers = os.path.join(module_dir, "Module.symvers")
    built_files = [built_kmod, built_symvers]
    for built_file in built_files:
        try:
            shutil.copy(built_file, output_dir)
        except (OSError, IOError) as e:
            msg = "Copy file from input path: {}\n" \
                  "to output path: {}" \
                  "finished with error: {}"
            logger.error(msg.format(built_file, output_dir, e))
            sys.exit(1)

if __name__ == "__main__":
    logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.WARNING)

    parser = argparse.ArgumentParser(description="Encapsulate an out-of-tree kernel module build, "
                                     "where the build does not modify the source directory")
    parser.add_argument("--output", "-o", required=True,
                        help="Kernel module to build (including output path)")
    parser.add_argument("--sources", "-s", metavar="FILE", nargs="+", required=True,
                        help="Kernel module source files")
    parser.add_argument("--depfile", "-d", metavar="DEPFILE", required=True,
                        help="Dependency file to generate")
    parser.add_argument("--common-root", "-r", default=None,
                        help="Common root directory that can be stripped from source paths")
    parser.add_argument("--module-dir", "-m",
                        help="Module output directory in kernel build")
    parser.add_argument("--jobs", "-j", metavar="N", default=None, type=int,
                        help="Allow N jobs at once")

    group = parser.add_argument_group("Kernel options")
    group.add_argument("--kernel", "-k", metavar="KDIR", required=True,
                       help="Kernel directory")
    group.add_argument("--cross-compile", "-c", default=None,
                       help="Kernel CROSS_COMPILE")
    group.add_argument("--kbuild-options", nargs="+", default=[],
                       help="Kernel config options to enable, that get added to EXTRA_CFLAGS too")
    group.add_argument("--extra-cflags", default="",
                       help="Options to add to EXTRA_CFLAGS as a string")
    group.add_argument("--extra-symbols", nargs="+", default=None,
                       help="Paths to Module.symvers for external symbols")
    group.add_argument("make_args", nargs=argparse.REMAINDER, default=[],
                       help="Make variables to be set")

    group = parser.add_argument_group("Dependency checking")
    group.add_argument("--include-dir", "-I", metavar="INCLUDE_DIR", action="append", default=[],
                       help="Include file search path")

    args = parser.parse_args()

    # The build is run in $KDIR, rather than the usual build workdir, so
    # parameters need to be absolute so they are accessible with a different CWD.
    output_dir = os.path.dirname(args.output)
    abs_output_dir = os.path.abspath(output_dir)
    abs_kdir = os.path.abspath(args.kernel)
    search_path = [os.path.abspath(d) for d in args.include_dir]

    # Don't abspath cross_compile if it's just a prefix for something already
    # inside $PATH (i.e. it doesn't contain a directory part):
    cross_compile = args.cross_compile
    if os.path.dirname(args.cross_compile):
        cross_compile = os.path.abspath(cross_compile)

    # Prepend EXTRA_CFLAGS with modified include paths
    includes = ["-I" + s for s in search_path]
    extra_cflags = " ".join(includes) + " " + args.extra_cflags + " " + \
                   " ".join([kbuild_to_cflag(o) for o in args.kbuild_options])

    deps = []

    # Add commonly needed search paths for copy_with_deps
    arch = kernel_config_parser.get_arch(abs_kdir)
    if not arch:
        sys.exit(1)

    search_path.extend([str.format(d, kdir=abs_kdir, arch=arch) for d in kernel_search_paths])
    kconfig = os.path.join(abs_kdir, "linux", "kconfig.h")
    for src_rel in args.sources:
        if args.common_root:
            dest = os.path.join(output_dir, os.path.relpath(src_rel, args.common_root))
        else:
            dest = os.path.join(output_dir, src_rel)
        deps.extend(copy_with_deps.copy_with_deps(src_rel, dest, search_path, [kconfig]))

    deps = sorted(set(deps))

    # Add a dependency on copy_with_deps.py, which won't have been set by Bob
    deps.append(os.path.join(os.path.dirname(sys.argv[0]), "copy_with_deps.py"))

    copy_with_deps.write_depfile(args.depfile, args.output, deps)

    make_args = args.make_args
    make_args.extend(args.kbuild_options)
    make_args.append("ARCH="+arch)
    if cross_compile:
        make_args.append("CROSS_COMPILE="+cross_compile)
    if args.extra_symbols is not None:
        extra_symbols = [os.path.abspath(d) for d in args.extra_symbols]
        make_args.append("KBUILD_EXTRA_SYMBOLS="+" ".join(extra_symbols))

    if args.jobs:
        make_args.append("-j"+str(args.jobs))
    else:
        # If the following env var is set, we are running in a build
        # farm where we should avoid increasing thread
        # count. Therefore leave make to run with a single core.
        # If not, build kernel modules with the number of CPUs we have.
        if os.getenv("MPDTI_BUILD_PARALLELISM") is None:
            make_args.append("-j"+str(multiprocessing.cpu_count()))

    module_ko = os.path.basename(args.output)
    abs_module_dir = os.path.abspath(args.module_dir)
    build_module(output_dir, module_ko, abs_kdir, abs_module_dir, make_args, extra_cflags)
