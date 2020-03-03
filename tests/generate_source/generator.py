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
import sys

parser = argparse.ArgumentParser(description='Test generator.')
parser.add_argument('--in', nargs='*', dest='input', action='store', help='Input files')
parser.add_argument('--out', nargs='*', dest='output', action='store', help='Output file')
parser.add_argument("--expect-in", nargs='*', action='store',
                    help='Basenames of expected input files')

args = parser.parse_args()

if args.expect_in:
    received_basenames = sorted(os.path.basename(i) for i in args.input)
    expected_basenames = sorted(args.expect_in)
    if received_basenames != expected_basenames:
        print("Expected the following files:", ", ".join(expected_basenames))
        print("But received these:", ", ".join(received_basenames))
        sys.exit(1)

for input_file in args.input:
    if not os.path.exists(input_file):
        print("Input file doesn't exist: " + input_file)
        sys.exit(1)

for out in args.output:
    file_name = os.path.basename(out)
    without_extension = os.path.splitext(file_name)[0]
    with open(out, 'w') as outfile:
        outfile.write("void output_%s(){}\n" % (without_extension))
