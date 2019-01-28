#!/usr/bin/env python

# Copyright 2018 Arm Limited.
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

import glob
import sys
import re
import os
import shutil
from argparse import ArgumentParser

# This script is actually within our package, so add the package to the python path
TEST_DIR = os.path.dirname(os.path.abspath(__file__))
BOB_DIR = os.path.dirname(os.path.dirname(TEST_DIR))
sys.path.append(BOB_DIR)


def runtest(name):
    from config_system import general
    print("Running %s" % name)

    tests_run = 0
    tests_failed = 0

    # Remove ".test" and replace with ".config" to find the input configuration
    config_file = name[:-5] + ".config"

    if not os.path.exists(config_file):
        config_file = "empty_file"
    general.read_config(name, config_file, False)

    with open(name) as f:
        line_number = 0
        for line in f:
            line_number += 1
            m = re.match("# (ASSERT|SET): (\S+)=(.+)", line)
            if m is None:
                continue
            action = m.group(1)
            (key, value) = (m.group(2), m.group(3))
            if action == "ASSERT":
                tests_run += 1
                actual_value = general.get_config(key).get("value")
                if actual_value != value:
                    print("ERROR: %s:%d: assertion failed: %s=%s (should be %s)"
                          % (name, line_number, key, actual_value, value))
                    tests_failed += 1
            elif action == "SET":
                general.set_config(key, value)
            else:
                raise Exception("Unexpected action %s" % action)

    return tests_run, tests_failed


def exe_conf_and_read_result(name):
    if os.system("conf --olddefconfig " + name):
        raise Exception("Failed to run 'conf': is it in your path?")

    config = {}
    with open(".config") as f:
        for line in f:
            m = re.match("CONFIG_(\S+)=(\S+)", line)
            if m is not None:
                config[m.group(1)] = m.group(2)
    return config


def runkconftest(name):
    print("Running %s" % name)

    tests_run = 0
    tests_failed = 0

    # Remove ".test" and replace with ".config" to find the input configuration
    config_file = name[:-5] + ".config"

    if not os.path.exists(config_file):
        config_file = "empty_file"

    shutil.copyfile(config_file, ".config")
    config = exe_conf_and_read_result(name)
    need_rerun = False

    with open(name) as f:
        line_number = 0
        for line in f:
            line_number += 1
            m = re.match("# (ASSERT|SET): (\S+)=(.+)", line)
            if m is None:
                continue
            action = m.group(1)
            (key, value) = (m.group(2), m.group(3))
            if action == "ASSERT":
                if need_rerun:
                    config = exe_conf_and_read_result(name)
                    need_rerun = False
                tests_run += 1
                actual_value = config.get(key, "n")
                if actual_value != value:
                    print("ERROR: %s:%d: assertion failed: %s=%s (should be %s)"
                          % (name, line_number, key, actual_value, value))
                    tests_failed += 1
            elif action == "SET":
                with open(".config", "a") as f:
                    f.write("CONFIG_%s=%s\n" % (key, value))
                need_rerun = True
            else:
                raise Exception("Unexpected action %s" % action)

    os.unlink(".config")
    os.unlink(".config.old")

    return tests_run, tests_failed


def main():
    parser = ArgumentParser()
    parser.add_argument("--kconf", dest="use_kconf", action="store_true", default=False,
                        help="Run using the kernel's configuration system."
                             "'conf' must be in the path")
    parser.add_argument("-C", "--directory", type=str, default=TEST_DIR)
    args = parser.parse_args()

    os.chdir(args.directory)

    tests_run = 0
    tests_failed = 0
    tests = glob.glob("*.test")
    for test in tests:
        if args.use_kconf:
            (run, failed) = runkconftest(test)
        else:
            (run, failed) = runtest(test)
        tests_run += run
        tests_failed += failed

    print("")
    print("%d tests run, %d failed" % (tests_run, tests_failed))
    if tests_failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
