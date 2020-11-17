#!/usr/bin/env python

# This confidential and proprietary software may be used only as
# authorised by a licensing agreement from ARM Limited
# (C) COPYRIGHT 2020 ARM Limited
# ALL RIGHTS RESERVED
# The entire notice above must be reproduced on all authorised
# copies and copies may only be made to the extent permitted
# by a licensing agreement from ARM Limited.

import argparse
import os
import subprocess
import sys


def generate_out_file(utility):
    return subprocess.check_output(utility).decode("utf-8")


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument('-i', '--input', required=True,
                        type=argparse.FileType('rt'), help="Input file")
    parser.add_argument('-o', '--out', required=True,
                        type=argparse.FileType('wt'), help="Output file name to be generated")
    parser.add_argument('-u', '--utility', required=True,
                        help="Binary utility to produce output")

    return parser.parse_args()


def main():
    args = parse_args()
    args.out.write(generate_out_file(args.utility))


if __name__ == "__main__":
    sys.exit(main())
