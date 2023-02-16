#!/usr/bin/env python3

# Copyright 2019-2020, 2022-2023 Arm Limited.
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

parser = argparse.ArgumentParser(description="Test generator outputing depfile")
parser.add_argument("input", default=[], action="append", help="Input file(s)")
parser.add_argument("-o", "--output", help="Output file")
parser.add_argument("-d", "--depfile", help="Dependency file")
parser.add_argument(
    "--gen-implicit-out",
    action="store_true",
    help="Flag to generate implicit output file",
)

args = parser.parse_args()

base = os.path.dirname(__file__)
implicit_ins = [os.path.join(base, "depgen2.in"), os.path.join(base, "depgen3.in")]

args.input += implicit_ins

with open(args.output, "w") as out:
    for input_file in args.input:
        if not os.path.exists(input_file):
            print("Input file doesn't exist: " + input_file)
            exit(-1)

        with open(input_file, "r") as f:
            out.write(f.read())

template = "{target}: {deps}\n"
dep_str = " \\\n\t".join(args.input)
with open(args.depfile, "w") as depfile:
    depfile.write(template.format(target=args.output, deps=dep_str))

# create empty file for test purposes, in the same folder as out file
if args.gen_implicit_out:
    outdir = os.path.dirname(args.output)
    with open(os.path.join(outdir, "out.h"), "w") as implicit_out:
        implicit_out.write("")
