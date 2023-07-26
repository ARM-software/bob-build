#!/usr/bin/env python3


import argparse
import os
import sys

parser = argparse.ArgumentParser(description="Test generator.")
parser.add_argument("--in", nargs="*", dest="input", action="store", help="Input files")
parser.add_argument(
    "--out", nargs="*", dest="output", action="store", help="Output file"
)
parser.add_argument(
    "--expect-in", nargs="*", action="store", help="Basenames of expected input files"
)
parser.add_argument("--config", nargs="?", action="store", help="config file")
parser.add_argument("--depfile", nargs="?", action="store", help="dependency file")

args = parser.parse_args()

if args.expect_in:
    received_basenames = sorted(os.path.basename(i) for i in args.input)
    expected_basenames = sorted(os.path.basename(i) for i in args.expect_in)
    if received_basenames != expected_basenames:
        print("Expected the following files:", ", ".join(expected_basenames))
        print("But received these:", ", ".join(received_basenames))
        sys.exit(1)

if args.depfile:
    template = "{target}: {deps}\n"
    dep_str = " \\\n\t".join(args.input)
    with open(args.depfile, "w") as depfile:
        depfile.write(template.format(target=args.output, deps=dep_str))

for input_file in args.input:
    if not os.path.exists(input_file):
        print("Input file doesn't exist: " + input_file)
        sys.exit(1)

for out in args.output:
    file_name = os.path.basename(out)
    without_extension = os.path.splitext(file_name)[0]
    with open(out, "w") as outfile:
        outfile.write("void output_%s(){}\n" % (without_extension))

if args.config:
    config_file = os.path.basename(args.config)
    if config_file != "bob.config":
        print("Wrong config file name: " + args.config)
        sys.exit(1)
