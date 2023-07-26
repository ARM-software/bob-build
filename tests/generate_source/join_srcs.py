#!/usr/bin/env python3


import argparse
import os
import sys

parser = argparse.ArgumentParser(
    description="Join input sources into single output file."
)
parser.add_argument(
    "--src-dir", action="store", required=True, help="Dir containing source files"
)
parser.add_argument("--use-a", action="store_true", help="Use file A as input")
parser.add_argument("--use-c", action="store_true", help="Use file C as input")
parser.add_argument(
    "--out", dest="output", action="store", required=True, help="Output file"
)
args = parser.parse_args()

inputs = []
if args.use_a:
    inputs.append(os.path.join(args.src_dir, "an.implicit.src"))
if args.use_c:
    inputs.append(os.path.join(args.src_dir, "cn.src"))

try:
    with open(args.output, "w") as outfile:
        for infile in inputs:
            if not os.path.exists(infile):
                print("Input file doesn't exist: " + infile)
                sys.exit(1)
            outfile.write(open(infile, "r").read() + "\n")
except IOError as e:
    print("Output file couldn't be created: " + str(e))
    sys.exit(1)
