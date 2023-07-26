#!/usr/bin/env python3


import argparse
import os


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", type=argparse.FileType("wt"), required=True)
    ap.add_argument("input", nargs="+", type=argparse.FileType("rt"))
    args = ap.parse_args()

    for i in args.input:
        if os.path.splitext(i.name)[1].lower() in (".c", ".cpp", ".cxx"):
            args.out.write(i.read())


if __name__ == "__main__":
    main()
