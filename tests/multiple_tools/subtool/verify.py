#!/bin/python

# Copyright 2022-2023 Arm Limited.
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
