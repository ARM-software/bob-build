#!/bin/env python

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

from __future__ import print_function

import argparse
import os
import shutil

SOURCE_CONTENT = """
int bob_test_implicit_out() {
    return 88;
}
"""

HEADER_CONTENT = """
#define A 88

"""


def main():
    parser = argparse.ArgumentParser(description="Test generator.")
    parser.add_argument("input", help="Input files")
    parser.add_argument("-o", "--output", help="Output file")
    parser.add_argument("--header", action="store_true",
                        help="Generate implicit header")
    parser.add_argument("--source", action="store_true",
                        help="Generate implicit source")

    implicit = "lib"
    args = parser.parse_args()
    inp = args.input
    out = args.output

    if not os.path.exists(inp):
        print("Input file doesn't exist: " + inp)
        exit(-1)

    shutil.copy(inp, out)

    path = os.path.join(os.path.dirname(out), implicit)

    if args.header:
        # create lib.h
        with open(path + ".h", "w") as hf:
            hf.write(HEADER_CONTENT)

    if args.source:
        # create lib.c
        with open(path + ".c", "w") as cf:
            cf.write(SOURCE_CONTENT)


if __name__ == "__main__":
    main()
