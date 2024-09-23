#!/usr/bin/env python3


import glob
import sys
import re
import os
import shutil
from argparse import ArgumentParser

# This script is actually within our package, so add the package to the python path
TEST_DIR = os.path.dirname(os.path.abspath(__file__))
CFG_DIR = os.path.dirname(TEST_DIR)
sys.path.append(CFG_DIR)

import config_system  # nopep8: E402 module level import not at top of file


def runtest(name):
    print("Running %s" % name)

    tests_run, tests_failed = 0, 0

    # Remove ".test" and replace with ".config" to find the input configuration
    config_file = os.path.splitext(name)[0] + ".config"

    if not os.path.exists(config_file):
        config_file = "empty_file"
    config_system.read_config(name, config_file, False)

    with open(name) as f:
        for line_number, line in enumerate(f):
            m = re.match(r"# (ASSERT|SET): (\S+)=(.+)", line)
            if not m:
                continue
            action, key, value = m.groups()
            if action == "ASSERT":
                tests_run += 1
                config = config_system.get_config(key)
                actual_value = config.get("value")
                if config["datatype"] == "bool":
                    actual_value = "y" if actual_value else "n"
                if actual_value != value:
                    print(
                        "ERROR: %s:%d: assertion failed: %s=%s (should be %s)"
                        % (name, line_number, key, actual_value, value)
                    )
                    tests_failed += 1
            elif action == "SET":
                config_system.set_config(key, value)
            else:
                raise Exception("Unexpected action %s" % action)

    return tests_run, tests_failed


def exe_conf_and_read_result(name):
    if os.system("conf --olddefconfig " + name):
        raise Exception("Failed to run 'conf': is it in your path?")

    config = {}
    with open(".config") as f:
        for line in f:
            m = re.match(r"CONFIG_(\S+)=(\S+)", line)
            if not m:
                continue
            config[m.group(1)] = m.group(2)
    return config


def runkconftest(name):
    print("Running %s" % name)

    tests_run, tests_failed = 0, 0

    # Remove ".test" and replace with ".config" to find the input configuration
    config_file = os.path.splitext(name)[0] + ".config"

    if not os.path.exists(config_file):
        config_file = "empty_file"

    shutil.copyfile(config_file, ".config")
    config = exe_conf_and_read_result(name)
    need_rerun = False

    with open(name) as f:
        for line_number, line in enumerate(f):
            m = re.match(r"# (ASSERT|SET): (\S+)=(.+)", line)
            if not m:
                continue
            action, key, value = m.groups()
            if action == "ASSERT":
                if need_rerun:
                    config = exe_conf_and_read_result(name)
                    need_rerun = False
                tests_run += 1
                actual_value = config.get(key, "n")
                if actual_value != value:
                    print(
                        "ERROR: %s:%d: assertion failed: %s=%s (should be %s)"
                        % (name, line_number, key, actual_value, value)
                    )
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
    parser.add_argument(
        "--kconf",
        dest="use_kconf",
        action="store_true",
        default=False,
        help="Run using the kernel's configuration system."
        "'conf' must be in the path",
    )
    parser.add_argument("-C", "--directory", type=str, default=TEST_DIR)
    args = parser.parse_args()

    os.chdir(args.directory)

    tests_run, tests_failed = 0, 0
    tests = glob.glob("*.test")
    test_runner = runkconftest if args.use_kconf else runtest
    for test in tests:
        run, failed = test_runner(test)
        tests_run += run
        tests_failed += failed

    print("")
    print("%d tests run, %d failed" % (tests_run, tests_failed))
    if tests_failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
