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

"""
Move debug files into the build ID structure that GDB can use.

This script assumes that note.gnu.build-id is available in the debug
file. You may need to set -Wl,--build-id on the link command line. For
more information see
https://sourceware.org/gdb/onlinedocs/gdb/Separate-Debug-Files.html
"""

from __future__ import print_function

import argparse
import errno
import os
import re
import shutil
import subprocess
import sys


# Regular expression to pick up Build ID from readelf output
RE_ID = re.compile(r"Build ID:\s+([a-f0-9]+)")


def get_build_id(f):
    cmd = ["readelf", "-n", f]
    try:
        with open(os.devnull, 'w') as devnull:
            output = subprocess.check_output(cmd, stderr=devnull)
            output = output.decode(sys.getdefaultencoding())
    except subprocess.CalledProcessError as e:
        sys.stderr.write("Error: Command %s failed with exit code %d" %
                         (str(cmd), e.returncode))
        sys.exit(e.returncode)

    # Look for Build ID
    for line in output.splitlines():
        m = RE_ID.search(line)
        if m:
            return m.group(1)

    return None


def make_dir(d):
    try:
        os.makedirs(d)
    except OSError as e:
        # Ignore errors if the dir already exists. Any other error number is
        # unexpected, so re-raise.
        if e.errno != errno.EEXIST:
            raise


def process_file(args, f):
    build_id = get_build_id(f)
    if build_id is not None:
        new_filedir = os.path.join(args.output, build_id[0:2])
        new_filename = os.path.join(new_filedir, build_id[2:]+".debug")
        if args.dry_run or args.verbose:
            print("Moving {} => {}".format(f, new_filename))
        if not(args.dry_run):
            make_dir(new_filedir)
            shutil.move(f, new_filename)

    elif args.dry_run or args.verbose:
        print("Not moving {}".format(f))


def parse_args():
    parser = argparse.ArgumentParser(epilog=__doc__,
                                     formatter_class=argparse.RawDescriptionHelpFormatter)

    parser.add_argument("input", nargs="+",
                        help="Path to input debug files. Directories will be assumed to "
                        "only contain debug files. Files will be handled individually.")
    parser.add_argument("-o", "--output", default="/usr/lib/debug/.build-id",
                        help="Target debug file directory")
    parser.add_argument("-n", "--dry-run", action="store_true",
                        help="Dry run")
    parser.add_argument("--verbose", action="store_true",
                        help="List all moves on console")

    return parser.parse_args()


def main():
    args = parse_args()

    for i in args.input:
        if os.path.isdir(i):
            for (dirpath, dirnames, filenames) in os.walk(args.input):
                for f in filenames:
                    f = os.path.join(dirpath, f)
                    process_file(args, f)
        elif os.path.isfile(i):
            process_file(args, i)
        else:
            sys.stderr.write("Error: {}: No such file or directory\n".format(i))
            sys.exit(1)


if __name__ == "__main__":
    main()
