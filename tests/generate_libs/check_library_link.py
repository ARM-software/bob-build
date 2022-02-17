#!/usr/bin/env python3

# Copyright 2020, 2022 Arm Limited.
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
import os
import re
import subprocess
import sys


def match_lines(cmd, regex):
    """Run `cmd` and return the match groups from each line matching `regex`"""
    try:
        dump = subprocess.check_output(cmd).decode()
    except subprocess.CalledProcessError as e:
        print(e)
        sys.exit(e.returncode)

    matched = set()
    for line in dump.splitlines():
        m = regex.match(line)
        if not m:
            continue
        matched.add(m.group(1))
    return matched


OBJDUMP_RE = re.compile(r'\s*NEEDED\s+([a-zA-Z0-9_-]+).so[0-9.]*')
OTOOL_RE = re.compile(r'\s+(.*?)(?:\.dylib)?\s+\(.*\)')

READ_DEPS_METHODS = {
    "objdump": lambda lib: match_lines(["objdump", "-p", lib], OBJDUMP_RE),
    "otool": lambda lib: [os.path.basename(i) for i in match_lines(["otool", "-L", lib], OTOOL_RE)],
}


def check_links(lib, links_to, read_deps):
    all_links = read_deps(lib)
    for link in links_to:
        if link not in all_links:
            print("ERROR: {} does not link to {}".format(lib, link))
            print("ERROR: The following dependencies were detected: " +
                  ", ".join(sorted(all_links)))
            sys.exit(1)


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument("--read-deps-method", choices=READ_DEPS_METHODS.keys(), default="objdump",
                    help="Program used to read library dependencies")
    ap.add_argument("--links-to", "-l", metavar="LIBNAME", default=[], action="append",
                    help="Check that LIBRARY links to LIBNAME")
    ap.add_argument("library")

    return ap.parse_args()


def main():
    args = parse_args()
    check_links(args.library, args.links_to, READ_DEPS_METHODS[args.read_deps_method])


if __name__ == "__main__":
    main()
