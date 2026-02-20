#!/usr/bin/env python3

import sys

out = sys.argv[1]
expect_ar = sys.argv[2]
expect_cc = sys.argv[3]
ar = sys.argv[4]
cc = sys.argv[5]

assert ar == expect_ar, "Expected ar '%s', got '%s'" % (expect_ar, ar)
assert cc == expect_cc, "Expected cc '%s', got '%s'" % (expect_cc, cc)

with open(out, "wt") as fp:
    fp.write("ar=%s cc=%s\n" % (ar, cc))
