#!/usr/bin/env python3


import argparse
import hashlib
import os
import sys


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument(
        "--hash",
        type=str,
        required=True,
        help="Combined hash of the build.bp and Mconfig files",
    )
    # We have to create an output file to stop the check being run on every
    # build. FileType("wt") will automatically create one, so there's no need
    # for any extra code.
    ap.add_argument(
        "--out",
        type=argparse.FileType("wt"),
        help="Dummy output file name. This is written, but not used",
    )
    ap.add_argument(
        "inputs",
        nargs="+",
        type=str,
        default=[],
        help="build.bp and Mconfig files to check",
    )

    return ap.parse_args()


def main():
    args = parse_args()

    combinedHash = hashlib.sha1()

    for src in args.inputs:
        fileHash = hashlib.sha1()
        with open(src, "rb") as fp:
            fileHash.update(fp.read())
        combinedHash.update(fileHash.digest())

    if combinedHash.hexdigest() != args.hash:
        lines = [
            "+--------------------------------------------------------------------+",
            "| WARNING: Android.bp is not up to date with build.bp/Mconfig files! |",
            "| WARNING:               Please regenerate Android.bp                |",
            "+--------------------------------------------------------------------+",
        ]
        for line in lines:
            sys.stderr.write(line + "\n")


if __name__ == "__main__":
    main()
