#!/usr/bin/env python

# Copyright 2018-2020 Arm Limited.
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
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BOB_DIR = os.path.dirname(SCRIPT_DIR)
CFG_DIR = os.path.join(BOB_DIR, "config_system")
sys.path.append(CFG_DIR)
import config_system  # nopep8: E402 module level import not at top of file

logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.ERROR)
logger = logging.getLogger(__name__)


# This script will read the config file and output a minimal json file
# containing just the user settable options.

def config_to_json(database_fname, config_fname, ignore_missing):
    config_system.read_config(database_fname, config_fname, ignore_missing)
    config_list = config_system.get_config_list()

    configs = dict()

    for key in config_list:
        c = config_system.get_config(key)
        key = key.lower()
        datatype = c['datatype']
        value = c['value']

        if 'title' in c and config_system.can_enable(c):
            if datatype in ['bool', 'string']:
                configs[key] = value
            elif datatype == 'int':
                configs[key] = int(value)
            else:
                msg = "Invalid config type: %s (with value '%s')\n"
                logger.critical(msg % (datatype, str(value)))
                sys.exit(1)

    return configs


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
    parser.add_argument("--depfile", type=argparse.FileType("wt"),
                        help="Write dependencies to a depfile")
    args = parser.parse_args()

    if not os.path.isfile(args.database):
        logger.error("No such file: %s" % args.database)
        sys.exit(1)
    if not os.path.isfile(args.config):
        logger.error("No such file: %s" % args.config)
        sys.exit(1)

    json_config = config_to_json(args.database, args.config, args.ignore_missing)
    with config_system.utils.open_and_write_if_changed(args.output) as fp:
        json.dump(json_config, fp, sort_keys=True, indent=4, separators=(',', ': '))

    if args.depfile:
        args.depfile.write("{output}: {config} {database}\n".format(**vars(args)))


if __name__ == "__main__":
    main()
