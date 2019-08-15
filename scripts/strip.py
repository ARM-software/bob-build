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


def create_debug_info(fname, dbg, objcopy):
    # Retain the build-id in the debug object
    cmd = [objcopy, "--only-keep-debug", fname, dbg]
    run(cmd)


def write_output(fname, output, dbg, strip, objcopy):
    cmd = [objcopy]
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
    parser.add_argument("--objcopy", default="objcopy",
                        help="Objcopy executable")

    args = parser.parse_args()

    return args


def main():
    args = parse_args()

    if args.debug_file:
        make_dir(os.path.dirname(args.debug_file))
        create_debug_info(args.input, args.debug_file, args.objcopy)

    write_output(args.input, args.output, args.debug_file, args.strip, args.objcopy)


if __name__ == "__main__":
    main()
