#!/usr/bin/env python3


import argparse
import os

parser = argparse.ArgumentParser(description="Test generator outputing depfile")
parser.add_argument("input", default=[], action="append", help="Input file(s)")
parser.add_argument("-o", "--output", help="Output file")
parser.add_argument("-d", "--depfile", help="Dependency file")
parser.add_argument(
    "--gen-implicit-out",
    action="store_true",
    help="Flag to generate implicit output file",
)

args = parser.parse_args()

base = os.path.dirname(__file__)
implicit_ins = [os.path.join(base, "depgen2.in"), os.path.join(base, "depgen3.in")]

args.input += implicit_ins

with open(args.output, "w") as out:
    for input_file in args.input:
        if not os.path.exists(input_file):
            print("Input file doesn't exist: " + input_file)
            exit(-1)

        with open(input_file, "r") as f:
            out.write(f.read())

template = "{target}: {deps}\n"
dep_str = " \\\n\t".join(args.input)
with open(args.depfile, "w") as depfile:
    depfile.write(template.format(target=args.output, deps=dep_str))

# create empty file for test purposes, in the same folder as out file
if args.gen_implicit_out:
    outdir = os.path.dirname(args.output)
    with open(os.path.join(outdir, "out.h"), "w") as implicit_out:
        implicit_out.write("")
