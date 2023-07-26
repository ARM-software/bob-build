#!/usr/bin/env python3


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
