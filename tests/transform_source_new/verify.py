#!/usr/bin/env python3

# Copyright 2023 Arm Limited.
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

from os import path


def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("input", action="store", help="Input file")
    ap.add_argument("output", action="store", help="Output file")
    ap.add_argument("expected", action="store", help="Expected output file")

    return ap, ap.parse_args()


def main():
    ap, args = parse_args()

    exp_file_base = path.basename(args.expected)
    out_file_base = path.basename(args.output)

    in_file = path.splitext(path.basename(args.input))[0]
    out_file = path.splitext(out_file_base)[0]

    if in_file != out_file:
        ap.error("Input & output names differ: '{}' != '{}'".format(in_file, out_file))

    if out_file_base != exp_file_base:
        ap.error(
            "Output names is incorrect: '{}' != '{}'".format(
                out_file_base, exp_file_base
            )
        )

    with open(args.output, "wt") as outfile:
        outfile.write("// Dummy output")


if __name__ == "__main__":
    main()
