#!/usr/bin/env python3

# Copyright 2020, 2022-2023 Arm Limited.
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


def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("out", action="store", help="Output file")

    return ap.parse_args()


def main():
    args = parse_args()

    with open(args.out, "wt") as outfile:
        outfile.write("void output_{func}() {{}}\n")


if __name__ == "__main__":
    main()
