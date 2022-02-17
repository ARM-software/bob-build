#!/usr/bin/env python3

import argparse
import os

SOURCE_TEMPLATE = """int {name}(void) {{
    return 0;
}}
"""

HEADER_TEMPLATE = "int {name}(void);\n"


def parse_args():
    ap = argparse.ArgumentParser()

    ap.add_argument("function_name")
    ap.add_argument("source")
    ap.add_argument("header")

    return ap.parse_args()


def main():
    args = parse_args()

    for fname, template in [(args.source, SOURCE_TEMPLATE),
                            (args.header, HEADER_TEMPLATE)]:
        with open(fname, "wt") as fp:
            fp.write(template.format(name=args.function_name))


if __name__ == "__main__":
    main()
