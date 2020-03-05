#!/bin/python

# Copyright 2018-2020 Arm Limited.
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


def basename_no_ext(fname):
    return os.path.splitext(os.path.basename(fname))[0]


def parse_args():
    ap = argparse.ArgumentParser(description='Test generator.')
    ap.add_argument("--in", dest="input", action="store", help="Input file", required=True)
    ap.add_argument("--gen-src", action="store", help="Source file to generate", required=True)
    ap.add_argument("--src-template", type=argparse.FileType("rt"),
                    help="Template file to use for source file generation")

    header = ap.add_mutually_exclusive_group(required=True)
    header.add_argument("--gen-header", action="store", help="Header to generate")
    header.add_argument("--gen-implicit-header", action="store_true",
                        help="Generate a header alongside the generated source")

    args = ap.parse_args()

    if args.gen_implicit_header:
        args.gen_header = os.path.join(os.path.dirname(args.gen_src),
                                       basename_no_ext(args.gen_src) + ".h")

    # Do some basic checks to ensure the transform source regexp replacement
    # worked as expected.
    if os.path.splitext(args.input)[1] != ".in":
        ap.error("Input file does not have `.in` extension: {}".format(args.input))

    if os.path.splitext(args.gen_src)[1] != ".cpp":
        ap.error("Generated source file does not have `.cpp` extension: {}".format(args.gen_src))

    if basename_no_ext(args.gen_header) != basename_no_ext(args.input):
        ap.error("Basename of generated output {} does not match input {}".format(args.gen_header,
                                                                                  args.input))

    if basename_no_ext(args.gen_src) != basename_no_ext(args.input):
        ap.error("Basename of generated output {} does not match input {}".format(args.gen_src,
                                                                                  args.input))

    if os.path.splitext(args.gen_header)[1] != ".h":
        ap.error("Generated header file does not have `.h` extension: {}".format(args.gen_src))

    return args


def main():
    args = parse_args()

    func = basename_no_ext(args.input)

    if args.src_template:
        src_template = args.src_template.read()
    else:
        src_template = "void output_{func}() {{}}\n"

    header_template = "void output_{func}();\n"

    with open(args.gen_src, 'wt') as outfile:
        outfile.write(src_template.format(func=func))

    with open(args.gen_header, 'wt') as outfile:
        outfile.write(header_template.format(func=func))


if __name__ == "__main__":
    main()
