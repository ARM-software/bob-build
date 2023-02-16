#!/bin/python

# Copyright 2022-2023 Arm Limited.
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


parser = argparse.ArgumentParser(description="Test generator.")
parser.add_argument("--in", dest="input", action="store", help="Input file")
parser.add_argument(
    "--out", nargs="*", dest="output", action="store", help="Output file"
)


def main():
    args = parser.parse_args()

    with open(args.input, "r") as f_in:
        for out in args.output:
            with open(out, "w") as f_out:
                f_out.write(f_in.read().replace("%%template%%", out))
                f_in.seek(0)


if __name__ == "__main__":
    main()
