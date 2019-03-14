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
import importlib
import logging
import os
import re
import sys

# This script is actually within our package, so add the package to the python path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from config_system.general import enforce_dependent_values, get_config, init_config, \
    read_config_file, read_profile_file, set_config_if_prompt, write_config
from config_system import log_handlers

root_logger = logging.getLogger()
root_logger.setLevel(logging.WARNING)

# Add counting Handler
counter = log_handlers.ErrorCounterHandler()
root_logger.addHandler(counter)

# Add StreamHandler with color Formatter
stream = logging.StreamHandler()
formatter = log_handlers.ColorFormatter("%(levelname)s: %(message)s", stream.stream.isatty())
stream.setFormatter(formatter)
root_logger.addHandler(stream)

logger = logging.getLogger(__name__)

parser = argparse.ArgumentParser()
parser.add_argument('-o', '--output', required=True,
                    help='Path to the output file')
parser.add_argument('-d', '--database', default="Mconfig",
                    help='Path to the configuration database (Mconfig)')
parser.add_argument('-p', '--plugin', action='append',
                    help='Post configuration plugin to execute',
                    default=[])
parser.add_argument('--ignore-missing', action='store_true', default=False,
                    help="Ignore missing database files included with 'source'")
parser.add_argument('args', nargs="*")
args = parser.parse_args()

init_config(args.database, args.ignore_missing)

files = []
setters = []

CONFIG_ARG_RE = re.compile(r'^([A-Za-z_][A-Za-z0-9_]*)=(.*)$')
def parse_config_arg(arg):
    """ Parse a KEY=VALUE command-line argument """
    m = CONFIG_ARG_RE.match(arg)
    if m is None:
        return None, None
    else:
        return m.group(1), m.group(2)

for arg in args.args:
    key, value = parse_config_arg(arg)
    if key:
        set_config_if_prompt(key, value, True)
    else:
        logger.info("Reading %s" % arg)
        read_profile_file(arg)

enforce_dependent_values()

for arg in args.args:
    key, value = parse_config_arg(arg)
    if key:
        try:
            final_value = get_config(key)['value']
        except KeyError:
            logger.error("unknown configuration option \"%s\"" % key)
        else:
            if final_value != value:
                logger.error("%s=%s was ignored or overriden. Value is %s" %
                             (key, value, final_value))

for plugin in args.plugin:
    path, name = os.path.split(plugin)
    if path.startswith('/'):
        sys.path.insert(0, path)
    else:
        sys.path.insert(0, os.path.join(os.getcwd(), path))
    sys.path.append(os.path.dirname(sys.argv[0]))
    try:
        mod = importlib.import_module(name)
        mod.plugin_exec()
    except ImportError as err:
        logger.error("Could not import %s plugin: %s" % (name, err))
    except Exception as err:
        logger.warning("Problem encountered in %s plugin: %s" % (name, repr(err)))
        import traceback
        traceback.print_tb(sys.exc_info()[2])

write_config(args.output)

issues = counter.errors() + counter.criticals()
warnings = counter.warnings()
if issues > 0:
    sys.exit(2)
elif warnings > 0:
    sys.exit(1)
else:
    sys.exit(0)
