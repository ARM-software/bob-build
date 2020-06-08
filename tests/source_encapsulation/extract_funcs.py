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
import sys
import re

parser = argparse.ArgumentParser(description='Extract function names and put into funcs.txt')
parser.add_argument('--in', dest='input', action='store', required=True, help='Input file')
parser.add_argument('--out', dest='output', action='store', required=True, help='Output file')
args = parser.parse_args()

# detects beginning of C-function signature
func_re = r'\S+\s+(\S+)\s*\(\S*\)\s*\{'

try:
    with open(args.input, 'r') as infile:
        res = re.findall(func_re, infile.read())
        try:
            with open(args.output, 'w') as outfile:
                outfile.write(' '.join(res))
        except IOError as e:
            print("Output file couldn't be created: " + str(e))
            sys.exit(1)
except IOError as e:
    print("Input file couldn't be opened: " + str(e))
    sys.exit(1)
