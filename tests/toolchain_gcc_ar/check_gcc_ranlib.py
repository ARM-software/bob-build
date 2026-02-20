#!/usr/bin/env python3

import sys

out = sys.argv[1]
expect_ranlib = sys.argv[2]
expect_cc = sys.argv[3]
ranlib = sys.argv[4]
cc = sys.argv[5]

assert ranlib == expect_ranlib, "Expected ranlib '%s', got '%s'" % (
    expect_ranlib,
    ranlib,
)
assert cc == expect_cc, "Expected cc '%s', got '%s'" % (expect_cc, cc)

with open(out, "wt") as fp:
    fp.write("ranlib=%s cc=%s\n" % (ranlib, cc))
