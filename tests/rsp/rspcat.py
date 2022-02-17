#!/usr/bin/env python3

import argparse


def ListFile(fname):
    fp = argparse.FileType("rt")(fname)
    return fp.read().split()


def main():
    ap = argparse.ArgumentParser()

    inputs = ap.add_mutually_exclusive_group(required=True)
    inputs.add_argument("--input", nargs="+", default=[])
    inputs.add_argument("--input_list", type=ListFile)

    outputs = ap.add_mutually_exclusive_group(required=True)
    outputs.add_argument("--output", nargs="+", default=[])
    outputs.add_argument("--output_list", type=ListFile)

    args = ap.parse_args()

    for output_file in args.output_list or args.output:
        with open(output_file, "wt") as out_fp:
            for input_file in args.input_list or args.input:
                with open(input_file, "rt") as in_fp:
                    out_fp.write(in_fp.read())


if __name__ == "__main__":
    main()
