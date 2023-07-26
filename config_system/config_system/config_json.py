#!/usr/bin/env python3


import json
import logging

from config_system import data, utils


logger = logging.getLogger(__name__)


def config_to_json():
    properties = dict()

    for key in data.get_config_list():
        c = data.get_config(key)
        key = key.lower()
        datatype = c["datatype"]
        value = c["value"]

        if datatype not in ["bool", "int", "string"]:
            logger.error(
                "Invalid config type: %s (with value '%s')\n" % (datatype, str(value))
            )

        if datatype == "int":
            value = int(value)

        properties[key] = {"ignore": c["bob_ignore"] == "y", "value": value}

    return properties


def write_config(filename):
    """Write configuration as a JSON file"""
    json_config = config_to_json()
    with utils.open_and_write_if_changed(filename) as fp:
        json.dump(json_config, fp, sort_keys=True, indent=4, separators=(",", ": "))
