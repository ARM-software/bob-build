#!/usr/bin/env python3

# Copyright 2022 Arm Limited.
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
import sys
import os
import platform

parser = argparse.ArgumentParser(description='Test generator.')
parser.add_argument('--out')
parser.add_argument('--expected')
group = parser.add_mutually_exclusive_group()
group.add_argument('--shared', help='use .so or .dylib extension', action='store_true')
group.add_argument('--static', help='use .a extension', action='store_true')

args = parser.parse_args()

if args.shared:
    if platform.system() == 'Darwin':
        extension = '.dylib'
    else:
        extension = '.so'
elif args.static:
    extension = '.a'
else:
    extension = ''

expected = args.expected + extension

if os.path.basename(args.out) != expected:
    print("Output from generation: {} but expected: {}".format(args.out, expected))
    sys.exit(1)
