#!/usr/bin/env python3

import argparse
import os
import shutil


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument(
        "--check-basename", nargs="+", action="append", metavar=("PATH ...", "BASE")
    )
    ap.add_argument("--copy", nargs="+", action="append", metavar=("SRC", "DEST"))

    return ap.parse_args()


def main():
    args = parse_args()

    for check in args.check_basename:
        # The first half of the arguments are paths, the second are basenames
        paths = check[0 : int(len(check) / 2)]
        basenames = check[int(len(check) / 2) :]
        assert len(paths) == len(
            basenames
        ), "All paths must have a corresponding basename"
        for path, basename in zip(paths, basenames):
            assert (
                os.path.basename(path) == basename
            ), "basename of '%s' is not equal to '%s'" % (path, basename)

    for copy in args.copy:
        assert len(copy) >= 2, "At least one source and destination required"
        srcs = copy[:-1]
        dest = copy[-1]
        assert len(srcs) == 1 or os.path.isdir(
            dest
        ), "Destination must be an existing directory when copying multiple sources"

        for src in srcs:
            shutil.copy(src, dest)


if __name__ == "__main__":
    main()
