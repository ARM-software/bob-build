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


def check_expected_input(input_files, expected_files):
    if len(input_files) != len(expected_files):
        print("Length mismatch! Input: {} Expected: {}".format(input_files, expected_files))
        sys.exit(1)

    for exp in expected_files:
        found = False
        for inp in input_files:
            if inp.endswith(exp):
                found = True
                break
        if not found:
            print("Missed expected file '{}' within input {}".format(exp, input_files))
            sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description='''Check whether provided input files match the \
                                                    expected ones. Generate fun3.c using input \
                                                    from funcs.txt''')
    parser.add_argument('--in', dest='input', nargs='+', default=[], required=True,
                        help='Input file list')
    parser.add_argument('--expected', dest='expected', default=[], nargs='+',
                        required=True, help='Expected input file list')
    parser.add_argument('--out', dest='output', action='store', required=True, help='Output file',
                        type=argparse.FileType('wt'))
    args = parser.parse_args()

    s = '''
#define FUNCS "%(funcs)s"
int fun3(void)
{
    return 0;
}
'''.lstrip()

    check_expected_input(args.input, args.expected)

    try:
        for f in args.input:
            filename = os.path.basename(f)
            if filename == "funcs.txt":
                with open(f, 'r') as infile:
                    d = {'funcs': infile.read()}
                    args.output.write((s % d) + '\n')
    except IOError as e:
        print("Input file couldn't be opened: " + str(e))
        sys.exit(1)


if __name__ == "__main__":
    main()
