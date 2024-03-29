#!/usr/bin/env python3


import argparse
import logging
import re
import sys

from config_system import log_handlers
from config_system.general import read_config, get_user_set_options

root_logger = logging.getLogger()
root_logger.setLevel(logging.WARNING)

# Add StreamHandler with color Formatter
stream = logging.StreamHandler()
formatter = log_handlers.ColorFormatter(
    "%(levelname)s: %(message)s", stream.stream.isatty()
)
stream.setFormatter(formatter)
root_logger.addHandler(stream)


def print_user_config():
    """
    Prints configuration which has been set by the user

    Particular options are grouped based on provided source.
    For those options with empty source group name is 'no source'
    """
    configs = {}
    for config in get_user_set_options():
        (key, value, source) = config
        if source not in configs:
            configs[source] = [(key, value)]
        else:
            configs[source].append((key, value))

    for key in configs:
        print("#\n# %s\n#" % (key if key else "no source"))
        for name, value in configs[key]:
            print("%s=%s" % (name, value))
        print("")


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-c",
        "--config",
        required=True,
        help="Path to the configuration file (*.config)",
    )
    parser.add_argument(
        "-d",
        "--database",
        default="Mconfig",
        help="Path to the configuration database (Mconfig)",
    )
    parser.add_argument(
        "--ignore-missing",
        action="store_true",
        default=False,
        help="Ignore missing database files included with 'source'",
    )
    return parser.parse_args()


def main():
    args = parse_args()

    read_config(args.database, args.config, args.ignore_missing)
    print_user_config()


if __name__ == "__main__":
    sys.exit(main())
