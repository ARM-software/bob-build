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

from config_system import log_handlers
from config_system.general import enforce_dependent_values, get_config, init_config, \
    read_profile_file, set_config_if_prompt, write_config, can_enable, format_dependency_list

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

CONFIG_ARG_RE = re.compile(r'^([A-Za-z_][A-Za-z0-9_]*)=(.*)$')


def parse_config_arg(arg):
    """ Parse a KEY=VALUE command-line argument """
    m = CONFIG_ARG_RE.match(arg)
    if m is None:
        return None, None
    else:
        return m.group(1), m.group(2)


def rindex(list, value):
    """Find the last-occurring index of `value` in `list`."""
    for i in range(len(list) - 1, 0, -1):
        if list[i] == value:
            return i
    return -1


def check_value_as_requested(key, requested_value, later_keys, later_values):
    try:
        opt = get_config(key)
    except KeyError:
        logger.error("unknown configuration option \"%s\"" % key)
        return

    final_value = opt["value"]

    if opt["datatype"] == "int":
        final_value = str(final_value)

    if final_value == requested_value:
        return

    # See if the argument was overridden by a later argument
    last_idx = rindex(later_keys, key)
    if last_idx != -1 and later_values[last_idx] != requested_value:
        logger.error("%s=%s was overridden by later argument %s=%s",
                     key, requested_value, key, later_values[last_idx])
        return

    depends = opt.get("depends")
    if depends and not can_enable(depends):
        logger.error("%s=%s was ignored; its dependencies were not met: %s",
                     key, requested_value, format_dependency_list(depends, skip_parens=True))
        return

    # Check this *after* dependencies. This allows users to investigate why an
    # option with unmet dependencies wasn't enabled, even if it isn't user-settable.
    if not opt.get("title"):
        logger.error("%s=%s was ignored; it has no title, so is not user-settable "
                     "(%s has no unmet dependencies)", key, requested_value, key)
        return

    logger.error("%s=%s was ignored or overriden. Value is '%s' %s %s",
                 key, requested_value, final_value, type(requested_value), type(final_value))


def check_values_as_requested(args):
    keys = []
    values = []
    for arg in args:
        key, value = parse_config_arg(arg)
        if key:
            keys.append(key)
            values.append(value)

    for i in range(0, len(keys)):
        key = keys[i]
        requested_value = values[i]
        check_value_as_requested(key, requested_value, keys[i+1:], values[i+1:])


def parse_args():
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
    return parser.parse_args()


def main():
    args = parse_args()

    init_config(args.database, args.ignore_missing)

    files = []
    setters = []

    for arg in args.args:
        key, value = parse_config_arg(arg)
        if key:
            set_config_if_prompt(key, value, True)
        else:
            logger.info("Reading %s" % arg)
            read_profile_file(arg)

    enforce_dependent_values()
    check_values_as_requested(args.args)

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
        return 2
    elif warnings > 0:
        return 1
    else:
        return 0


if __name__ == "__main__":
    sys.exit(main())
