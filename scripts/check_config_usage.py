#!/usr/bin/env python3

# Copyright 2022 Arm Limited.
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
import fnmatch
import os
import re
import sys


class Configs:
    _configs = dict()

    # Support the in keyword
    def __contains__(self, key):
        return key in self._configs

    # Record a new config
    def append(self, config, file, line):
        self._configs[config] = {'file': file,
                                 'line': line}

    # Return a list of all configs
    def keys(self):
        return self._configs.keys()

    # Return the file where a config is defined
    def file(self, config):
        return self._configs[config]['file']

    # Print an issue to stdout
    def print_issue(self, type, config, file, line):
        print("%s,%s,%d,%s,%d,%s" % (config, file, line, self._configs[config]['file'],
                                     self._configs[config]['line'], type))


# Find files matching pattern under the directory top
# Returns a list of filenames (which include top as prefix)
def find_files(top, pattern):
    matches = []
    for root, dirnames, filenames in os.walk(top):
        for filename in fnmatch.filter(filenames, pattern):
            matches.append(os.path.join(root, filename))
    return matches


# b is in the same subtree as a
#
# Assumes no symlinks. If a and b are relative, they are from the same
# starting location.
def same_subtree(a, b):
    dira = os.path.dirname(a)
    dirb = os.path.dirname(b)

    return dirb.startswith(dira)


# Collect all definitions in all Mconfigs
# Returns a dictionary with each config as key
def get_config_definitions(mconfigs):
    configs = Configs()

    re_configdef = re.compile(r"^config ([A-z_][A-z0-9_]*)")

    for mconfig in mconfigs:
        with open(mconfig, 'r') as f:
            lineno = 0
            for line in f:
                lineno += 1
                m = re_configdef.match(line)
                if m:
                    key = m.group(1)
                    if key in configs:
                        msg = "Error %s already defined in %s. Also in %s\n"
                        sys.stderr.write(msg % (key, configs.file(key), mconfig))
                    else:
                        configs.append(key, mconfig, lineno)

    return configs


# Look for all occurrences of defined configs in Mconfig files.
def check_mconfig_refs(mconfigs, configs):
    keyre = str.join('|', configs.keys())
    re_configs = re.compile(r"\b(" + keyre + ")\b")
    for mconfig in mconfigs:
        with open(mconfig, 'r') as f:
            lineno = 0
            for line in f:
                lineno += 1

                # Strip out single line comments
                line = line.split('#')[0]

                matches = re_configs.findall(line)
                for key in matches:
                    if not same_subtree(configs.file(key), mconfig):
                        configs.print_issue('Mconfig', key, mconfig, lineno)


# Look for all occurrences of defined configs in Blueprint files.
# In blueprint the reference can only occur in a feature or a template
def check_blueprint_refs(blueprints, configs):
    keyre = str.join('|', configs.keys())
    keyre = keyre.lower()
    re_featureref = re.compile(r"[ \t]*(" + keyre + ")[ \t]*:")
    re_templateref = re.compile(r"{{(?:.+ *)?\.(" + keyre + ")}}")
    for blueprint in blueprints:
        with open(blueprint, 'r') as f:
            lineno = 0
            for line in f:
                lineno += 1

                # Strip out single line comments
                line = line.split('//')[0]

                m = re_featureref.match(line)
                if m:
                    key = m.group(1).upper()
                    if not same_subtree(configs.file(key), blueprint):
                        configs.print_issue('bp feature', key, blueprint, lineno)

                matches = re_templateref.findall(line)
                for key in matches:
                    key = key.upper()
                    if not same_subtree(configs.file(key), blueprint):
                        configs.print_issue('bp template', key, blueprint, lineno)


def main():
    summary = \
        """
        Detect usage of config variables outside of the tree they are
        defined in.  This script can be used to help detect potential
        issues if subtrees can be excluded.
        """
    epilog = \
        """
        The output is comma separated text of with a header row describing
        the fields. The header row starts with '#' to allow sorting.
        """

    parser = argparse.ArgumentParser(description=summary, epilog=epilog)
    parser.add_argument('path', nargs='?', default=os.getcwd(), help="Directory to scan")
    parser.add_argument('--nom', default=False, action='store_true',
                        help="Don't check Mconfig files")
    parser.add_argument('--nob', default=False, action='store_true',
                        help="Don't check Blueprint files")

    args = parser.parse_args()

    check_bp = not args.nob
    check_mc = not args.nom

    # Find all Mconfig files
    mconfigs = find_files(args.path, 'Mconfig')

    # Table header
    print("# Config, Referenced from, Ref line, Defined in, Def line, Type")

    # Find all configs and where they are defined
    defs = get_config_definitions(mconfigs)

    if check_mc:
        # Check Mconfigs for where configs are referenced
        check_mconfig_refs(mconfigs, defs)

    if check_bp:
        # Find all bp files
        blueprints = find_files(args.path, '*.bp')

        # Check bps for where configs are referenced
        check_blueprint_refs(blueprints, defs)


if __name__ == "__main__":
    main()
