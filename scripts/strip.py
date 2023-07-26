#!/usr/bin/env python3


import argparse
import errno
import os
import subprocess
import sys


def make_dir(d):
    try:
        os.makedirs(d)
    except OSError as e:
        # Ignore errors if the dir already exists. Any other error number is
        # unexpected, so re-raise.
        if e.errno != errno.EEXIST:
            raise


def run(cmd):
    try:
        subprocess.check_call(cmd)
    except subprocess.CalledProcessError as e:
        sys.stderr.write(
            "Error: Command %s failed with exit code %d" % (str(cmd), e.returncode)
        )
        sys.exit(e.returncode)
    except OSError as e:
        sys.stderr.write(
            "Error: Couldn't execute command '%s': %s" % (" ".join(cmd), e.strerror)
        )
        sys.exit(1)


def elf_create_debug_info(fname, dbg, tool):
    # Retain the build-id in the debug object
    cmd = [tool, "--only-keep-debug", fname, dbg]
    run(cmd)


def macho_create_debug_info(fname, dbg, tool):
    cmd = [tool, fname, "-o", dbg]
    run(cmd)


def elf_write_output(fname, output, dbg, strip, tool):
    cmd = [tool]
    if dbg:
        cmd.extend(["--strip-debug", "--add-gnu-debuglink=" + dbg])
    if strip:
        cmd.append("--strip-unneeded")
    cmd.extend([fname, output])

    run(cmd)


def macho_write_output(fname, output, dbg, strip, tool):
    run([tool, "-u", "-o", output, fname])


def parse_args():
    parser = argparse.ArgumentParser()

    parser.add_argument("input", help="Library/executable to strip")
    parser.add_argument("-o", "--output", required=True, help="Stripped file")
    parser.add_argument(
        "--strip",
        action="store_true",
        default=False,
        help="Strip library of unnecessary symbols",
    )
    parser.add_argument("--debug-file", default=None, help="File to keep debug info in")
    parser.add_argument(
        "--format",
        action="store",
        choices=["elf", "macho"],
        default="elf",
        help="Library format",
    )
    parser.add_argument(
        "--objcopy-tool",
        default="objcopy",
        help="Tool to use with Elf libraries, including path if needed. "
        "This is expected to be objcopy on Linux platforms",
    )
    parser.add_argument(
        "--dsymutil-tool",
        default="dsymutil",
        help="Tool used to manipulate debug info with Mach-O libraries, "
        "including path if needed. "
        "This is expected to be dsymutil on OSX",
    )
    parser.add_argument(
        "--strip-tool",
        default="strip",
        help="Tool used to strip Mach-O libraries, including path if needed."
        "This is expected to be strip on OSX",
    )

    args = parser.parse_args()

    return args


def main():
    args = parse_args()

    if args.format == "macho":
        create_debug_info = macho_create_debug_info
        write_output = macho_write_output
        debug_info_tool = args.dsymutil_tool
        strip_tool = args.strip_tool
    else:
        create_debug_info = elf_create_debug_info
        write_output = elf_write_output
        debug_info_tool = args.objcopy_tool
        strip_tool = args.objcopy_tool

    if args.debug_file:
        make_dir(os.path.dirname(args.debug_file))
        create_debug_info(args.input, args.debug_file, debug_info_tool)

    write_output(args.input, args.output, args.debug_file, args.strip, strip_tool)


if __name__ == "__main__":
    main()
