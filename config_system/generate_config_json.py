#!/usr/bin/env python3

# Copyright 2019, 2022 Arm Limited.
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
import logging
import sys

import config_system
import config_system.log_handlers
import config_system.config_json

root_logger = logging.getLogger()
root_logger.setLevel(logging.WARNING)

# Add counting Handler
counter = config_system.log_handlers.ErrorCounterHandler()
root_logger.addHandler(counter)


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument('config',
                        help="Path to the configuration file (*.config)")
    parser.add_argument('-d', '--database', default="Mconfig",
                        help='Path to the configuration database (Mconfig)')
    parser.add_argument('--ignore-missing', action='store_true', default=False,
                        help="Ignore missing database files included with 'source'")
    parser.add_argument('-j', '--json', metavar="OUT", required=True,
                        help="Write JSON configuration file")
    return parser.parse_args()


def main():
    args = parse_args()

    config_system.read_config(args.database, args.config, args.ignore_missing)

    config_system.config_json.write_config(args.json)

    return counter.errors() + counter.criticals()


if __name__ == "__main__":
    sys.exit(main())
