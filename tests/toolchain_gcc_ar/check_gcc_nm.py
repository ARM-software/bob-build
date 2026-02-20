#!/usr/bin/env python3

import sys

out = sys.argv[1]
expect_nm = sys.argv[2]
expect_cc = sys.argv[3]
nm = sys.argv[4]
cc = sys.argv[5]

assert nm == expect_nm, "Expected nm '%s', got '%s'" % (expect_nm, nm)
assert cc == expect_cc, "Expected cc '%s', got '%s'" % (expect_cc, cc)

with open(out, "wt") as fp:
    fp.write("nm=%s cc=%s\n" % (nm, cc))
