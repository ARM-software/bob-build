#!/usr/bin/env python3


import argparse
import sys
import os
import platform

parser = argparse.ArgumentParser(description="Test generator.")
parser.add_argument("--out")
parser.add_argument("--expected")
group = parser.add_mutually_exclusive_group()
group.add_argument("--shared", help="use .so or .dylib extension", action="store_true")
group.add_argument("--static", help="use .a extension", action="store_true")

args = parser.parse_args()

if args.shared:
    if platform.system() == "Darwin":
        extension = ".dylib"
    else:
        extension = ".so"
elif args.static:
    extension = ".a"
else:
    extension = ""

expected = args.expected + extension

if os.path.basename(args.out) != expected:
    print("Output from generation: {} but expected: {}".format(args.out, expected))
    sys.exit(1)
