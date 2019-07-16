# Copyright 2018-2019 Arm Limited.
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

from config_system import general, utils


logger = logging.getLogger(__name__)


def config_to_json():
    properties = dict()

    for key in general.get_config_list():
        c = general.get_config(key)
        key = key.lower()
        datatype = c["datatype"]
        value = c["value"]

        if datatype == "bool":
            value = True if value == "y" else False
            properties[key] = value
        elif datatype == "int":
            properties[key] = int(value)
        elif datatype == "string":
            properties[key] = value
        else:
            logger.error("Invalid config type: %s (with value '%s')\n" % (datatype, str(value)))

    return properties


def write_config(filename):
    """Write configuration as a JSON file"""
    json_config = config_to_json()
    with utils.open_and_write_if_changed(filename) as fp:
        json.dump(json_config, fp, sort_keys=True, indent=4, separators=(",", ": "))
