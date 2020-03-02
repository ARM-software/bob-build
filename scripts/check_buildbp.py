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

import argparse
import hashlib
import os
import sys


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument("--hash", type=str, required=True,
                    help="Combined hash of build.bp files")
    # We have to create an output file to stop the check being run on every
    # build. FileType("wt") will automatically create one, so there's no need
    # for any extra code.
    ap.add_argument("--out", type=argparse.FileType("wt"),
                    help="Dummy output file name. This is written, but not used")
    ap.add_argument("inputs", nargs="+", type=str, default=[],
                    help="build.bp files to check")

    return ap.parse_args()


def main():
    args = parse_args()

    combinedHash = hashlib.sha1()

    for buildbp in sorted(args.inputs):
        fileHash = hashlib.sha1()
        with open(buildbp, "rb") as fp:
            fileHash.update(fp.read())
        combinedHash.update(fileHash.digest())

    if combinedHash.hexdigest() != args.hash:
        lines = [
            "+---------------------------------------------------------------------------+",
            "| WARNING: build.bp files have been changed since Android.bp was generated! |",
            "| WARNING:                  Please regenerate Android.bp                    |",
            "+---------------------------------------------------------------------------+",
        ]
        for line in lines:
            sys.stderr.write(line + "\n")


if __name__ == "__main__":
    main()
