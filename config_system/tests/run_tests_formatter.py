#!/usr/bin/env python

# Copyright 2019 Arm Limited.
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
import os
import sys
import tempfile

# Get file directory path
TEST_DIR = os.path.dirname(os.path.abspath(__file__))
BOB_DIR = os.path.dirname(os.path.dirname(TEST_DIR))
sys.path.append(BOB_DIR)


def run_test(name, expected_output):
    """ Test function to verify difference between two file contents"""
    from config_system import mconfigfmt

    passed = True

    print("Running %s" % name)
    tmp_file = tempfile.NamedTemporaryFile(mode="w+", delete=False)
    mconfigfmt.perform_formatting(name, tmp_file.file)
    tmp_file.close()

    with open(tmp_file.name) as test_out, open(expected_output) as exp_out:
        out_lines, exp_lines = test_out.readlines(), exp_out.readlines()
    if any(out_lines[i] != exp_lines[i] for i in range(len(exp_lines))):
        passed = False

    os.remove(tmp_file.name)

    return passed


def main():
    formatter_tests = os.path.join(TEST_DIR, "formatter")

    tests_passed = 0
    tests_failed = 0

    for fname in os.listdir(formatter_tests):
        base, ext = os.path.splitext(fname)
        if ext == ".test":
            test = os.path.join(formatter_tests, base)
            passed = run_test(test + ".test", test + ".expected")
            if passed:
                tests_passed += 1
            else:
                tests_failed += 1

    print("")
    print("{} tests run, {} failed".format(tests_passed + tests_failed, tests_failed))
    if tests_failed > 0:
        sys.exit(1)


if __name__ == '__main__':
    main()
