#!/usr/bin/env python

# Copyright 2019 Arm Limited.
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
import os
import re
import sys
import argparse

# Get file directory path
sys.path.append(os.path.dirname(os.path.dirname(os.path.realpath(__file__))))

from config_system.lex_wrapper import LexWrapper


def perform_formatting(file_list_path, output=None):
    """Call LexWrapper class to call PLY lexer facade, then get back outcome with formatting principles
    :param file_list_path: Input file path list
    :param output (optional) output file path
    """
    for file_path in file_list_path:
        wrapper = LexWrapper(ignore_missing=False, verbose=True)
        wrapper.source(file_path)
        dump(output or file_path, wrapper)


def dump(file_path, wrapper):
    """General method for dumping the output into file from given path"""
    line, lines = [], []
    for token in wrapper.iterate_tokens():
        if not token:
            break
        if token.type == "HELPTEXT" and len(token.value) > 1:
            line.append(handle_help_format(token))
            continue
        if "\n" not in str(token.value):
            line.append(handle_formatting(token))
            continue
        line.append(token.value)
        str_line = "".join(line)
        lines.append(str_line[:-1].rstrip(' ') + str_line[-1])
        line = []
    with open(file_path, "w") as f:
        f.writelines(lines)


def handle_formatting(token):
    """Handle formatting for various types of tokens"""
    if isinstance(token.value, int):
        return str(token.value)
    elif token.type == "QUOTED_STRING":
        return '"{}"'.format(token.value)
    return "{}".format(token.value)


def handle_help_format(token):
    """Workaround for indentation requirement"""
    help_ind = "  "
    indent, text = re.search(r"(\s+)(.+\n)", token.value).groups()
    # replace unfolded spaces to tabs
    indent = indent.replace("    ", "\t")
    if not indent.endswith(help_ind) or indent.count(' ') != len(help_ind):
        # replace last tab as 2 spaces and invalid amount of spaces
        indent = indent[:-1].rstrip(" ") + help_ind
    return "".join([indent, text])


def main():
    """Main function of formatter. Adds parser facade with two params input and output file
    Also checks via CheckPath action if file is present under given path.
    Input file need to be present and output file should not be present
    """
    parser = argparse.ArgumentParser(formatter_class=argparse.HelpFormatter)
    parser.add_argument('input', nargs='+',
                        help="Input file with configuration database (Mconfig) to fix.")
    args = parser.parse_args()
    perform_formatting(args.input)


if __name__ == "__main__":
    main()
