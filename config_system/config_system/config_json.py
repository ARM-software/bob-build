#!/usr/bin/env python3

# Copyright 2018-2022, 2021 Arm Limited.
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

import json
import logging
import os

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
            logger.error("Invalid config type: %s (with value '%s')\n" % (datatype, str(value)))

        if datatype == "int":
            value = int(value)

        properties[key] = {
            "ignore": c["bob_ignore"] == 'y',
            "value": value
        }

    return properties


def write_config(filename):
    """Write configuration as a JSON file"""
    json_config = config_to_json()
    with utils.open_and_write_if_changed(filename) as fp:
        json.dump(json_config, fp, sort_keys=True, indent=4, separators=(",", ": "))
