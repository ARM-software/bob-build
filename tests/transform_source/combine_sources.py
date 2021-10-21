#!/usr/bin/env python3

# Copyright 2021 Arm Limited.
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
