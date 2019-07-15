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

import hashlib
import json
import logging
import os

from config_system import general, utils


logger = logging.getLogger(__name__)


def hash_env():
    m = hashlib.sha256()
    for k in sorted(os.environ.keys()):
        val = os.environ[k]
        # When Python 2 is used make sure that utf-8 encoding is used to prevent non-ASCII errors
        if hasattr(os.environ[k], 'decode'):
            val = val.decode('utf-8')
        m.update(u"{}={}\n".format(k, val).encode('utf-8'))
    return m.hexdigest()


def config_to_json():
    features = dict()
    properties = dict()

    for key in general.get_config_list():
        c = general.get_config(key)
        key = key.lower()
        datatype = c["datatype"]
        value = c["value"]

        if datatype == "bool":
            value = True if value == "y" else False
            features[key] = value
            properties[key] = value
        elif datatype == "int":
            properties[key] = int(value)
        elif datatype == "string":
            properties[key] = value
        else:
            logger.error("Invalid config type: %s (with value '%s')\n" % (datatype, str(value)))

    properties["__bob_env_hash__"] = hash_env()

    return {"Features": features, "Properties": properties}


def write_config(filename):
    """Write Bob-specific JSON file"""
    json_config = config_to_json()
    with utils.open_and_write_if_changed(filename) as fp:
        json.dump(json_config, fp, sort_keys=True, indent=4, separators=(",", ": "))
