#!/bin/python


from __future__ import print_function

import argparse
import errno
import os


parser = argparse.ArgumentParser(description="Test generator.")
parser.add_argument("--in", nargs="*", dest="input", action="store", help="Input file")


def main():
    args = parser.parse_args()

    for in_f in args.input:
        if not (os.path.exists(in_f) and os.path.isfile(in_f)):
            raise OSError(errno.ENOENT, os.strerror(errno.ENOENT), in_f)


if __name__ == "__main__":
    main()
