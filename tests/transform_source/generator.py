#!/bin/python

# Copyright 2018 Arm Limited.
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

parser = argparse.ArgumentParser(description='Test generator.')
parser.add_argument('--in', dest='input', action='store', help='Input files')
parser.add_argument('--out', dest='output', action='store', help='Output file')

args = parser.parse_args()

if len(args.input) < 1 or len(args.output) < 1:
    print("Invalid input/output")
    print("Input: " + args.input)
    print("Output: " + args.output)
    exit(1)

input_basename = os.path.basename(args.input)
if os.path.splitext(input_basename)[1] != ".in":
    print("Input file should end with '.in': " + input_basename)
    exit(1)

output_base_name = os.path.basename(args.output)

if os.path.splitext(input_basename)[0] != os.path.splitext(output_base_name)[0]:
    print("Not matching base name: ")
    print("Input: ")
    print(os.path.splitext(input_basename))
    print("Output: ")
    print(os.path.splitext(output_base_name))
    exit(1)

if os.path.splitext(output_base_name)[1] != ".cpp":
    print("Output file should end with '.cpp': " + output_base_name)
    exit(1)

without_extension = os.path.splitext(output_base_name)[0]
with open(args.output, 'w') as outfile:
    outfile.write("void output_%s(){}\n" % without_extension)
