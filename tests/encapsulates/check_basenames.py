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


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument("--out", "-o", type=str, help="File to write on success")
    ap.add_argument("--expected", default=[], nargs="+", action="append", help="Expected basenames")
    ap.add_argument("--actual", default=[], nargs="+", action="append", help="Actual filenames")

    args = ap.parse_args()

    if len(args.expected) != len(args.actual):
        ap.error("Mismatch of expected ({}) vs actual ({}) lists".format(len(args.expected),
                                                                         len(args.actual)))

    return args


def check_basenames(expected, actual):
    """Check that each path suffix in `expected` has a corresponding path in `actual`"""
    if len(expected) != len(actual):
        print("Mismatching list lengths: expected {} ({}), got {} ({})".format(
              len(expected), str(expected), len(actual), str(actual)))
        return False

    passed = True

    for expected_basename in sorted(expected, key=len, reverse=True):
        expected_basename = os.sep + expected_basename  # Ensure we match complete path components
        found = False
        for i in range(0, len(actual)):
            actual_path = os.sep + actual[i]
            if actual_path.endswith(expected_basename):
                found = True
                del actual[i]
                break
        if not found:
            print("Could not find path containing expected basename '{}' in {}".format(
                  expected_basename[1:], str(actual)))
            passed = False

    return passed


def main():
    args = parse_args()

    passed = True

    for i in range(0, len(args.expected)):
        passed &= check_basenames(args.expected[i], args.actual[i])

    if not passed:
        if os.path.isfile(args.out):
            os.unlink(args.out)
        sys.exit(1)

    with open(args.out, "wt"):
        pass


if __name__ == "__main__":
    main()
