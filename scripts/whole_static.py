#!/usr/bin/env python3

# Copyright 2022 Arm Limited.
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from __future__ import print_function
from collections import defaultdict

import argparse
import os
import shutil
import subprocess
import sys
import tempfile


def list_archive_contents(archive):
    cmd = ["ar", "t", archive]
    out = subprocess.check_output(cmd)
    contents = out.decode("utf-8").splitlines()
    return [i.strip() for i in contents]


def extract_archive(ar, dest, archive):
    contents = list_archive_contents(archive)
    # Some linkers put extra files inside the archive, which should be ignored.
    contents = [i for i in contents if os.path.splitext(i)[1] == ".o"]

    # Archive files can contain duplicate files.
    # If this is the case we can't extract at once
    if len(contents) == len(set(contents)):
        cmd = [ar, "x", os.path.relpath(archive, dest)]
        subprocess.call(cmd, cwd=dest)
        return [os.path.join(dest, i) for i in contents]
    else:
        extracted_objects = defaultdict(lambda: 1)
        extracted_files = []
        for c in contents:
            work_dir = dest
            if c in extracted_objects:
                work_dir = os.path.join(dest, "%s.%d" % (c, extracted_objects[c]))
                os.mkdir(work_dir)
            archive_relative_to_dest = os.path.relpath(archive, work_dir)
            cmd = [ar, "xN", str(extracted_objects[c]), archive_relative_to_dest, c]
            subprocess.call(cmd, cwd=work_dir)
            extracted_objects[c] += 1
            extracted_files.append(os.path.join(work_dir, c))
        return extracted_files


def extract_archives(ar, dest, archives):
    extracted_objects = []

    for a in archives:
        this_dest_name = os.path.splitext(os.path.basename(a))[0]
        this_dest = os.path.join(dest, this_dest_name)
        os.mkdir(this_dest)
        extracted_objects += extract_archive(ar, this_dest, a)

    return extracted_objects


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument("--build-wrapper", required=False)
    ap.add_argument("--ar", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("inputs", nargs="+")

    return ap.parse_args()


def main():
    args = parse_args()

    objects = []
    archives = []

    # Always remove the output file to avoid appending to an existing archive.
    if os.path.isfile(args.out):
        try:
            os.remove(args.out)
        except OSError as e:
            sys.stderr.write(
                "Error: Couldn't remove file '%s': %s\n",
                args.out, e.strerror)
            sys.exit(1)

    for i in args.inputs:
        ext = os.path.splitext(i)[1]
        if ext == ".o":
            objects.append(i)
        elif ext == ".a":
            archives.append(i)
        else:
            sys.stderr.write(
                "Error: %s is not an object file or archive.\n" % i)
            sys.exit(1)

    args_out_dirname, args_out_basename = os.path.split(args.out)
    tmpdir = tempfile.mkdtemp(dir=args_out_dirname, prefix=args_out_basename + ".",
                              suffix=".tmp.d")

    try:
        extracted_objects = extract_archives(args.ar, tmpdir, archives)
        cmd = [args.ar, "-rcs", args.out] + objects + extracted_objects
        # prepend with build wrapper
        # note: we need to split as it can contain wrapper args as well
        if args.build_wrapper is not None:
            cmd = args.build_wrapper.split() + cmd
        subprocess.call(cmd)
    except subprocess.CalledProcessError as e:
        sys.stderr.write("Error: Command '%s' failed.\n" % e.cmd)
        sys.exit(e.returncode)
    except OSError as e:
        sys.stderr.write(
                "Error: Couldn't execute command '%s': %s\n" %
                (' '.join(cmd), e.strerror))
        sys.exit(1)
    finally:
        shutil.rmtree(tmpdir)


if __name__ == "__main__":
    main()
