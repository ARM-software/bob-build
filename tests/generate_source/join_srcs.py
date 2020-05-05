#!/usr/bin/env python

# Copyright 2020 Arm Limited.
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

parser = argparse.ArgumentParser(description='Join input sources into single output file.')
parser.add_argument('--src-dir', action='store', required=True, help='Dir containing source files')
parser.add_argument('--use-a', action='store_true', help='Use file A as input')
parser.add_argument('--use-c', action='store_true', help='Use file C as input')
parser.add_argument('--out', dest='output', action='store', required=True, help='Output file')
args = parser.parse_args()

inputs = []
if args.use_a:
    inputs.append(os.path.join(args.src_dir, 'an.implicit.src'))
if args.use_c:
    inputs.append(os.path.join(args.src_dir, 'cn.src'))

try:
    with open(args.output, 'w') as outfile:
        for infile in inputs:
            if not os.path.exists(infile):
                print("Input file doesn't exist: " + infile)
                sys.exit(1)
            outfile.write(open(infile, 'r').read() + '\n')
except IOError as e:
    print("Output file couldn't be created: " + str(e))
    sys.exit(1)
