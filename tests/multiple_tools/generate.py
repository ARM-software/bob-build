#!/bin/python


from __future__ import print_function

import argparse


parser = argparse.ArgumentParser(description="Test generator.")
parser.add_argument("--in", dest="input", action="store", help="Input file")
parser.add_argument(
    "--out", nargs="*", dest="output", action="store", help="Output file"
)


def main():
    args = parser.parse_args()

    with open(args.input, "r") as f_in:
        for out in args.output:
            with open(out, "w") as f_out:
                f_out.write(f_in.read().replace("%%template%%", out))
                f_in.seek(0)


if __name__ == "__main__":
    main()
