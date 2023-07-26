#!/usr/bin/env python3


from __future__ import print_function
import argparse


def parse_args():
    ap = argparse.ArgumentParser()
    ap.add_argument("out", action="store", help="Output file")

    return ap.parse_args()


def main():
    args = parse_args()

    with open(args.out, "wt") as outfile:
        outfile.write("void output_{func}() {{}}\n")


if __name__ == "__main__":
    main()
