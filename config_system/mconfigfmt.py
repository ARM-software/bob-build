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
import sys
import argparse

# Get file directory path
sys.path.append(os.path.dirname(os.path.dirname(os.path.realpath(__file__))))

from config_system.lex_wrapper import LexWrapper


def perform_formatting(file_path, output):
    """Call LexWrapper class to call PLY lexer facade, then get back outcome with formatting principles
    :param file_path: Input file path
    :param output: handle to file/stdout or file path (only if original file in use)
    """
    wrapper = LexWrapper(ignore_missing=False, verbose=True)
    wrapper.source(file_path)
    rewrite = isinstance(output, str)  # If string supplied -> assume file path
    if rewrite:
        output = open(output, "w")
    for token in wrapper.iterate_tokens():
        output.write(handle_formatting(token))
    if rewrite:
        output.close()


def handle_formatting(token):
    """Function which applies additional formatting to token value
    :return: Token value with changes to value if needed"""
    dec_map = {
        "BOOL": "\t{}".format,
        "INT": "\t{}".format,
        "STRING": "\t{}".format,
        "DEFAULT": "\t{}".format,
        "DEPENDS": "\t{}".format,
        "HELP": "\t{}".format,
        "SELECT": "\t{}".format,
        "PROMPT": "\t{}".format,
        "VISIBLE": "\t{}".format,
        "HELPTEXT": format_helptext,
        "QUOTED_STRING": '"{}"'.format,
    }
    handler = dec_map.get(token.type, str)
    return handler(token.value)


def format_helptext(value):
    """Handle formatting for HELPTEXT field.
    Apply formatting only for token with value otherwise supply with newline"""
    if not value:
        return "\n"
    return "\t  {}\n".format(value)


def main():
    """Main function of formatter. Adds parser facade with two params input and output file
    Also checks via CheckPath action if file is present under given path.
    Input file need to be present and output file should not be present
    """
    parser = argparse.ArgumentParser(formatter_class=argparse.HelpFormatter)
    parser.add_argument("input", nargs="+",
                        help="Input file with configuration database (Mconfig) to fix.")
    parser.add_argument("--write", "-w", default=False, action="store_true",
                        help="Write formatted output to original file")
    parser.add_argument("-o", "--output", help="Output file path")
    args = parser.parse_args()

    output_handle = open(args.output, "w") if args.output else sys.stdout
    for input_path in args.input:
        formatting_output = output_handle
        if args.write:
            formatting_output = input_path
        perform_formatting(input_path, formatting_output)
    output_handle.close()


if __name__ == "__main__":
    main()
