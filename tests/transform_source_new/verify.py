#!/usr/bin/env python3


from __future__ import print_function

import argparse

from os import path


def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("input", action="store", help="Input file")
    ap.add_argument("output", action="store", help="Output file")
    ap.add_argument("expected", action="store", help="Expected output file")

    return ap, ap.parse_args()


def main():
    ap, args = parse_args()

    exp_file_base = path.basename(args.expected)
    out_file_base = path.basename(args.output)

    in_file = path.splitext(path.basename(args.input))[0]
    out_file = path.splitext(out_file_base)[0]

    if in_file != out_file:
        ap.error("Input & output names differ: '{}' != '{}'".format(in_file, out_file))

    if out_file_base != exp_file_base:
        ap.error(
            "Output names is incorrect: '{}' != '{}'".format(
                out_file_base, exp_file_base
            )
        )

    with open(args.output, "wt") as outfile:
        outfile.write("// Dummy output")


if __name__ == "__main__":
    main()
