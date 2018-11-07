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

import json
import os.path
import sys
import logging

# The config system is in the directory above, so add it to the python path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import config_system

logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.ERROR)
logger = logging.getLogger(__name__)

# This script will read the config file and output a minimal json file
# containing just the user settable options.

def generate_config_json(database_fname, config_fname, ignore_missing):
    config_system.read_config(database_fname, config_fname, ignore_missing)
    config_list = config_system.get_config_list()

    configs = dict()

    for key in config_list:
        c = config_system.get_config(key)
        key = key.lower()
        datatype = c["datatype"]
        value = c["value"]

        if "title" in c and config_system.can_enable(c.get('depends')):
            if datatype == "bool":
                value = True if value == "y" else False
                configs[key] = value
            elif datatype == "int":
                configs[key] = int(value)
            elif datatype == "string":
                configs[key] = value
            else:
                logger.critical("Invalid config type: %s (with value '%s')\n" % (datatype, str(value)))
                sys.exit(1)

    return json.dumps(configs,
                      sort_keys=True, indent=4, separators=(',', ': '))


# Write 'text' to file 'fname' only if the content will change.
def write_if_different(fname, text):
    same_content = False
    if os.path.isfile(fname):
        with open(fname, "r+") as fp:
            original = fp.read()
            same_content = text == original

    if not same_content:
        logger.info("Writing config JSON to %s" % fname)
        with open(fname, "w") as fp:
            fp.write(text)


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
        logger.error("No such file: %s" % args.database)
        sys.exit(1)
    if not os.path.isfile(args.config):
        logger.error("No such file: %s" % args.config)
        sys.exit(1)

    text = generate_config_json(args.database, args.config, args.ignore_missing)
    write_if_different(args.output, text)


if __name__ == "__main__":
    main()
