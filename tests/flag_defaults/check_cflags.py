#!/usr/bin/env python

# Copyright 2018 Arm Limited.
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

import sys

out = sys.argv[1]
target_type = sys.argv[2]
flags = sys.argv[3:]

if target_type == "--check-host":
    assert "x86_64-linux-gnu" in flags
elif target_type == "--check-target":
    assert "aarch64-linux-gnu" in flags
else:
    assert False, "Invalid target type: '%s'" % target_type

assert "-DROOT_VAR=1" in flags
assert "-DSECOND_VAR=2" in flags

with open(out, "wt") as fp:
    fp.write(" ".join(flags))
