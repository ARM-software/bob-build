#!/usr/bin/env python

# Copyright 2019 Arm Limited.
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
import errno
import os
import subprocess
import sys


def make_dir(d):
    try:
        os.makedirs(d)
    except OSError as e:
        # Ignore errors if the dir already exists. Any other error number is
        # unexpected, so re-raise.
        if e.errno != errno.EEXIST:
            raise


def run(cmd):
    try:
        subprocess.check_call(cmd)
    except subprocess.CalledProcessError as e:
        sys.stderr.write("Error: Command %s failed with exit code %d" %
                         (str(cmd), e.returncode))
        sys.exit(e.returncode)
    except OSError as e:
        sys.stderr.write("Error: Couldn't execute command '%s': %s" % (' '.join(cmd), e.strerror))
        sys.exit(1)


def create_debug_info(fname, dbg, tool):
    # Retain the build-id in the debug object
    if tool == "dsymutil":
        cmd = [tool, fname, "-o", dbg]
    else:
        cmd = [tool, "--only-keep-debug", fname, dbg]
    run(cmd)


def write_output(fname, output, dbg, strip, tool):
    if os.path.basename(tool) == "dsymutil":
        run(["strip", "-u", "-o", output, fname])
    else:
        cmd = [tool]
        if dbg:
            cmd.extend(["--strip-debug",
                        "--add-gnu-debuglink=" + dbg])
        if strip:
            cmd.append("--strip-unneeded")
        cmd.extend([fname, output])

        run(cmd)


def parse_args():
    parser = argparse.ArgumentParser()

    parser.add_argument("input", help="Library/executable to strip")
    parser.add_argument("-o", "--output", required=True, help="Stripped file")
    parser.add_argument("--strip", action="store_true", default=False,
                        help="Strip library of unnecessary symbols")
    parser.add_argument("--debug-file", default=None,
                        help="File to keep debug info in")
    parser.add_argument("--tool", default="objcopy",
                        help="Primary tool to use for stripping (including path if needed)."
                             "This is expected to be objcopy on Linux platforms,"
                             "and dsymutil on MacOS.")

    args = parser.parse_args()

    return args


def main():
    args = parse_args()

    if args.debug_file:
        make_dir(os.path.dirname(args.debug_file))
        create_debug_info(args.input, args.debug_file, args.tool)

    write_output(args.input, args.output, args.debug_file, args.strip, args.tool)


if __name__ == "__main__":
    main()
