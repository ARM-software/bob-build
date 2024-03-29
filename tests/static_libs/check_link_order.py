#!/usr/bin/env python3


# This script is intended to check the link order for sl_main_dd,
# whose library dependencies look like:
#
#       main
#     /     \
#    c       d
#  /   \   /   \
# e     f g     h
#      /       /
#     g       e

import argparse
import os
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("cmd")
parser.add_argument("args", nargs=argparse.REMAINDER)

args = parser.parse_args()

# These are the dependencies that need to be satisfied
# Include the dependencies implicit in the ordering specified by top level static_libs
deps = {
    "sl_libc.a": ["sl_libd.a", "sl_libe.a", "sl_libf.a"],
    "sl_libd.a": ["sl_libg.a", "sl_libh.a"],
    "sl_libf.a": ["sl_libg.a"],
    "sl_libh.a": ["sl_libe.a"],
    # Implicit dependencies in static_libs are not followed
    # "sl_libe.a" : ["sl_libf.a"],
    # "sl_libg.a" : ["sl_libh.a"],
    "sl_libe.a": [],
    "sl_libg.a": [],
}

libs = []
compile_obj = False
# Pick up static libraries (*.a)
for arg in args.args:
    if arg == "-c":
        compile_obj = True
    (base, ext) = os.path.splitext(arg)
    if ext == ".a":
        basename = os.path.basename(arg)
        libs.append(basename)

error = False
if not compile_obj:
    # This should be a link command

    # For each library check that its dependencies occur after it
    # Libraries are allowed to be listed more than once (though we would prefer not to)
    for idx, lib in enumerate(libs):
        if lib in deps:
            for dep in deps[lib]:
                if dep not in libs[idx + 1 :]:
                    print("Error:", dep, "not after", lib)
                    error = True

    # Check every library is listed
    for lib in deps:
        if lib not in libs:
            print("Error:", lib, "missing")
            error = True

cmd = [args.cmd] + args.args
result = subprocess.call(cmd)

if result == 0 and error:
    # Compiler was OK, but lib order check is not
    exit(1)
else:
    # No error so pass on compiler return code
    exit(result)
