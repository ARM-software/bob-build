#!/usr/bin/env python

# Copyright 2018-2021 Arm Limited.
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

# The config system is in the directory above, so add it to the python path
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BOB_DIR = os.path.dirname(SCRIPT_DIR)
CFG_DIR = os.path.join(BOB_DIR, "config_system")
sys.path.append(CFG_DIR)

import config_system.utils as utils  # nopep8: E402 module level import not at top of file


def hash_env():
    """Hash only relevant environment options"""

    # List of relevant options which might influence the generation of
    # build.ninja and should be only taken into account while hashing
    #
    # For detailed information please check:
    #      gcc - https://gcc.gnu.org/onlinedocs/gcc/Environment-Variables.html
    #    clang - https://clang.llvm.org/docs/CommandGuide/clang.html#environment
    # armclang - https://developer.arm.com/documentation/100748/0606/Supporting-reference-information/Toolchain-environment-variables # nopep8: E501 line to long
    #
    relevant_env = [
        "LD_LIBRARY_PATH",
        "PATH",

        # bob-build
        "BOB_ALWAYS_LINK_SHARED_LIBS",
        "BOB_BOOTSTRAP_VERSION",
        "BOB_CONFIG_OPTS",
        "BOB_CONFIG_PLUGIN_OPTS",
        "BOB_CPUPROFILE",
        "BOB_DIR",
        "BOB_LINK_PARALLELISM",
        "BOB_VERSION",
        "BUILDDIR",
        "CONFIG_FILE",
        "CONFIG_JSON",
        "SRCDIR",
        "TOPNAME",
        "WORKDIR",

        # go
        "GO386",
        "GOARCH",
        "GOARM",
        "GOOS",
        "GOMIPS",
        "GOPATH",
        "GOROOT",

        # gcc
        "C_INCLUDE_PATH",
        "COMPILER_PATH",
        "CPATH",
        "CPLUS_INCLUDE_PATH",
        "DEPENDENCIES_OUTPUT",
        "GCC_COMPARE_DEBUG",
        "GCC_EXEC_PREFIX",
        "LIBRARY_PATH",
        "OBJC_INCLUDE_PATH",
        "SOURCE_DATE_EPOCH",
        "SUNPRO_DEPENDENCIES",

        # clang
        "MACOSX_DEPLOYMENT_TARGET",
        "OBJCPLUS_INCLUDE_PATH",

        # armclang
        "ARM_PRODUCT_PATH",
        "ARM_TOOL_VARIANT",
        "ARMCOMPILER6_ASMOPT",
        "ARMCOMPILER6_CLANGOPT",
        "ARMCOMPILER6_FROMELFOPT",
        "ARMCOMPILER6_LINKOPT",
        "ARMROOT"
    ]

    m = hashlib.sha256()
    for k in sorted(os.environ.keys()):
        if k in relevant_env:
            val = os.environ[k]
            # When Python 2 is used make sure that utf-8 encoding
            # is used to prevent non-ASCII errors
            if hasattr(os.environ[k], 'decode'):
                val = val.decode('utf-8')
            m.update(u"{}={}\n".format(k, val).encode('utf-8'))
    return m.hexdigest()


def write_env_hash(filename):
    """Write a hash of the current environment to the named file."""
    with utils.open_and_write_if_changed(filename) as fp:
        fp.write(hash_env())


def test_hash_env_relevant():
    """Test if relevant environment option will change the hash"""
    no_path_env = False
    old_path_env = ""
    org_hash = hash_env()

    # PATH is one of the option that bob cares about
    if "PATH" in os.environ.keys():
        old_path_env = os.environ['PATH']
    else:
        no_path_env = True

    # change PATH
    os.environ['PATH'] = "/new/fake/bob/directory"

    # hash should change
    assert org_hash != hash_env()

    # restore previous state and check hash
    if no_path_env:
        del os.environ['PATH']
    else:
        os.environ['PATH'] = old_path_env

    # hash should be the same again
    assert org_hash == hash_env()


def test_hash_env_irrelevant():
    """Test if irrelevant environment option will not change the hash"""
    org_hash = hash_env()

    # set FAKE_ENV environment option which should not change the hash
    os.environ['FAKE_ENV'] = "/fake/environment/set"

    # hash should not change
    assert org_hash == hash_env()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("output",
                        help="Output file to write containing environment hash")
    args = parser.parse_args()

    write_env_hash(args.output)


if __name__ == "__main__":
    main()
