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
import logging
import os
import re
import subprocess
import sys


logger = logging.getLogger(__name__)

"""
Record the significant parts of shared library content. When
anything here changes, the callers of the library must be relinked.

This includes the SONAME and the dynamic symbol table (ignoring
addresses and sizes).

This is expected to work on Linux and OSX.
"""

# Environment to use for processes we parse output from.
# Force the C locale.
child_env = os.environ.copy()
child_env['LC_ALL'] = "C"


def parse_args():
    parser = argparse.ArgumentParser(
        description="Generate table of contents for a shared library")
    parser.add_argument("-o", "--output", default=None,
                        help=".toc file to create")
    parser.add_argument("--format", action="store",
                        choices=["elf", "macho"], default="elf",
                        help="Library format")
    parser.add_argument("--objdump-tool", default="objdump",
                        help="Tool used to generate TOCs for Elf libraries. "
                             "This is expected to be objdump on Linux platforms")
    parser.add_argument("--otool-tool", default="otool",
                        help="Tool used to read library section headers of Mach-O libraries. "
                             "This is expected to be otool on OSX")
    parser.add_argument("--nm-tool", default="nm",
                        help="Tool used to read the dynamic symbol table of Mach-O libraries. "
                             "This is expected to be nm on OSX")
    parser.add_argument("input", help="Shared library")
    args = parser.parse_args()

    return args


def line_filter(regexp, lines):
    """
    Filter each line of input by a regular expression.

    If the regular expression is not matched, then the line is not output.
    """
    output = []
    for line in lines:
        match = regexp.match(line)
        if match:
            output.append(line)

    return output


def line_filter_and_transform(filter_re, transform_re, repl, lines):
    """
    Transform and filter lines.

    Filter out lines that match filter_re.

    Transform remaining lines by transform_re and repl as a regular
    expression replace. If the regular expression is not matched, then
    the line is output as is.
    """
    output = []
    for line in lines:
        match = filter_re.match(line)
        if match:
            # Drop line matching filter_re
            continue
        output.append(transform_re.sub(repl, line))

    return output


def elf_toc(lib, tool):
    """
    Generate a table of contents for ELF files.

    This function uses objdump from GNU binutils.
    """
    toc = []

    # Get private (format specific) headers, which includes SONAME
    cmd = [tool, "-p", lib]
    try:
        result = subprocess.check_output(cmd, env=child_env)
    except subprocess.CalledProcessError as e:
        logger.error("Command failed: %s", str(e.cmd))
        sys.exit(e.returncode)

    result_arr = result.decode(sys.getdefaultencoding()).split("\n")

    # `objdump -p` outputs a header per line, and some version information.
    # Just pick up the lines containing the symbols we're interested in.
    regexp = re.compile(r'\s+SONAME\s')
    toc.extend(line_filter(regexp, result_arr))

    # Get dynamic symbol table from objdump
    cmd = [tool, "-T", lib]
    try:
        result = subprocess.check_output(cmd, env=child_env)
    except subprocess.CalledProcessError as e:
        logger.error("Command failed: %s", str(e.cmd))
        sys.exit(e.returncode)

    result_arr = result.decode(sys.getdefaultencoding()).split("\n")

    # `objdump -T` outputs something like:
    #
    # install/lib/library.so:     file format elf64-x86-64
    #
    # DYNAMIC SYMBOL TABLE:
    # 0000000000000000  w   D  *UND*  0000000000000000 OPT_VER __gmon_start__
    # 0000000000000480 g    DF .init  0000000000000000         _init
    # 00000000000005dc g    DF .fini  0000000000000000         _fini
    # 0000000000001868 g    D  *ABS*  0000000000000000         _edata
    # 0000000000001868 g    D  *ABS*  0000000000000000         __bss_start
    # 0000000000001869 g    D  *ABS*  0000000000000000         _end
    #
    # The first column is address
    # The next 'column' are 7 flags (which may not be present)
    # The third column is the section name
    # The fourth is size.
    # After this is an optional version, and then the symbol
    #
    # We want to drop address and size
    #
    # See https://sourceware.org/binutils/docs/binutils/objdump.html
    #
    # Filter out undefined symbols, indicated with *UND* as the section name
    flags_re = r'[lgu! ][w ][C ][W ][Ii ][dD ][FfO ]'
    lax_flags_re = r'.{7}'
    section_re = r'\S+'
    hexdigits_re = r'[\da-f]+'

    filter_undefined_re = re.compile(hexdigits_re + r'\s' + lax_flags_re + r'\s\*UND\*')
    transform_re = re.compile(r'^' + hexdigits_re + r'\s(' +
                              flags_re + r'\s' +
                              section_re + r')\s+' +
                              hexdigits_re + r'(\s+.+)$')
    repl = r'\1\2'

    toc.extend(line_filter_and_transform(filter_undefined_re, transform_re, repl, result_arr))

    return toc


def macho_toc(lib, otool, nm):
    """
    Generate a table of contents for Mach-O format libraries.

    This relies on otool and nm. Don't currently support cross
    compiles.
    """
    toc = []

    # In Mach-O, the equivalent of SONAME is LC_ID_DYLIB. We can
    # retrieve this with `otool -D`
    cmd = [otool, "-D", lib]
    try:
        result = subprocess.check_output(cmd, env=child_env)
    except subprocess.CalledProcessError as e:
        logger.error("Command failed: %s", str(e.cmd))
        sys.exit(e.returncode)

    result_arr = result.decode(sys.getdefaultencoding()).split('\n')
    toc.extend(result_arr)

    # Get global symbols, portable format
    cmd = [nm, "-gP", lib]
    try:
        result = subprocess.check_output(cmd, env=child_env)
    except subprocess.CalledProcessError as e:
        logger.error("Command failed: %s", str(e.cmd))
        sys.exit(e.returncode)

    result_arr = result.decode(sys.getdefaultencoding()).split('\n')

    # The output of `nm -gP` is 4 columns: symbol, type, address?, size?
    # Only keep the first 2 columns, and drop undefined symbols (type 'U')
    filter_re = re.compile(r'\S+\sU\s')
    transform_re = re.compile(r'^(\S+\s[UATDBC\-SI])\s.*')
    repl = r'\1'

    toc.extend(line_filter_and_transform(filter_re, transform_re, repl, result_arr))

    return toc


def write_if_changed(filename, data):
    """
    Write data to file replacing current content, but only if the
    content has changed, or the file doesn't exist.
    """
    same_content = False
    try:
        if os.path.isfile(filename):
            with open(filename, "rt") as fp:
                original_content = fp.read()
                same_content = data == original_content
    finally:
        if not same_content:
            logger.debug("Updating {}".format(filename))
            with open(filename, "wt") as fp:
                fp.write(data)


def main():
    logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.WARNING)

    args = parse_args()

    if args.format == "elf":
        lines = elf_toc(args.input, args.objdump_tool)
    elif args.format == "macho":
        lines = macho_toc(args.input, args.otool_tool, args.nm_tool)

    toc = "\n".join(lines)
    toc += "\n"            # Include a newline at the end of the file
    if args.output:
        write_if_changed(args.output, toc)
    else:
        sys.stdout.write(toc)


if __name__ == "__main__":
    main()
