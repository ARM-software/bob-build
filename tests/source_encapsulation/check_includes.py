#!/usr/bin/env python

# Copyright 2019-2021 Arm Limited.
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

# This script is intended to check the generated deps order of
# 'validate_source_encapsulation_mod_deps' whose encapsulation look like:
#
#                    3 (root module)
#    (encapsulates) /
#                  2
#   (generated_deps) /
#                1
#

from __future__ import print_function

import os
import sys
import argparse
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument("cmd")
parser.add_argument("-I", action="append", dest="includes")
parser.add_argument("-c", action='store_true', dest="compile")

# Get only needed to test args
args, _ = parser.parse_known_args()

if not args.compile:
    # Terminate not related to testing commands
    exit(subprocess.call(sys.argv[1:]))

includes = [os.path.basename(i) for i in args.includes]

wanted_deps = [
    "gen_srcs_two",
    "encapsulation_source_mod_deps",
]
unwanted_deps = [
    "gen_srcs_one",
]

have_unwanted = any(inc in unwanted_deps for inc in includes)
have_wanted = all(inc in wanted_deps for inc in includes)

if have_unwanted:
    print("Unwanted dependencies have been included.")
    exit(1)
if not have_wanted:
    print("Missing required dependencies")
    exit(1)

exit(subprocess.call(sys.argv[1:]))
