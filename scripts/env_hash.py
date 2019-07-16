#!/usr/bin/env python

# Copyright 2018-2019 Arm Limited.
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
    m = hashlib.sha256()
    for k in sorted(os.environ.keys()):
        val = os.environ[k]
        # When Python 2 is used make sure that utf-8 encoding is used to prevent non-ASCII errors
        if hasattr(os.environ[k], 'decode'):
            val = val.decode('utf-8')
        m.update(u"{}={}\n".format(k, val).encode('utf-8'))
    return m.hexdigest()


def write_env_hash(filename):
    """Write a hash of the current environment to the named file."""
    with utils.open_and_write_if_changed(filename) as fp:
        fp.write(hash_env())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("output",
                        help="Output file to write containing environment hash")
    args = parser.parse_args()

    write_env_hash(args.output)


if __name__ == "__main__":
    main()
