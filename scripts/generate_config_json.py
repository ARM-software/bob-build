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

from __future__ import print_function

import hashlib
import json
import logging
import os
import sys

# The config system is in the directory above, inside package config system,
# so add it to the python path
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BOB_DIR = os.path.dirname(SCRIPT_DIR)
CFG_DIR = os.path.join(BOB_DIR, "config_system")
sys.path.append(CFG_DIR)
import config_system  # nopep8: E402 module level import not at top of file

logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.WARNING)


# This script will read the config file and output a json file containing the
# types and values of all config options, which will be read by Bob.

def generate_config_json(database_fname, config_fname, ignore_missing):
    config_system.read_config(database_fname, config_fname, ignore_missing)
    config_list = config_system.get_config_list()

    features = dict()
    properties = dict()

    for key in config_list:
        c = config_system.get_config(key)
        key = key.lower()
        datatype = c["datatype"]
        value = c["value"]

        if datatype == "bool":
            value = True if value == "y" else False
            features[key] = value
            properties[key] = value
        elif datatype == "int":
            properties[key] = int(value)
        elif datatype == "string":
            properties[key] = value
        else:
            sys.stderr.write("Invalid config type: %s (with value '%s')\n" % (datatype, str(value)))
            sys.exit(1)

    return features, properties


# Write 'text' to file 'fname' only if the content will change.
def write_if_different(fname, text):
    same_content = False
    if os.path.isfile(fname):
        with open(fname, "r+") as fp:
            original = fp.read()
            same_content = text == original

    if not same_content:
        print("Writing config JSON to %s" % fname)
        with open(fname, "w") as fp:
            fp.write(text)


def hash_env():
    m = hashlib.sha256()
    for k in sorted(os.environ.keys()):
        val = os.environ[k]
        # When Python 2 is used make sure that utf-8 encoding is used to prevent non-ASCII errors
        if hasattr(os.environ[k], 'decode'):
            val = val.decode('utf-8')
        m.update(u"{}={}\n".format(k, val).encode('utf-8'))
    return m.hexdigest()


def main():
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("config", help="Path to the configuration file (*.config)")
    parser.add_argument("-d", "--database", default="Mconfig",
                        help="Path to the configuration database (Mconfig)")
    parser.add_argument("-o", "--output", required=True,
                        help="Path to the config JSON file")
    parser.add_argument("--ignore-missing", default=False, action="store_true",
                        help="Ignore missing database files included with 'source'")
    args = parser.parse_args()

    if not os.path.isfile(args.database):
        print("Error: No such file: %s" % args.database)
        sys.exit(1)
    if not os.path.isfile(args.config):
        print("Error: No such file: %s" % args.config)
        sys.exit(1)

    features, properties = generate_config_json(args.database, args.config, args.ignore_missing)
    properties["__bob_env_hash__"] = hash_env()

    text = json.dumps({"Features": features, "Properties": properties},
                      sort_keys=True, indent=4, separators=(',', ': '))
    write_if_different(args.output, text)


if __name__ == "__main__":
    main()
