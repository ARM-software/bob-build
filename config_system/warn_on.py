#!/usr/bin/env python

# Copyright 2018 Arm Limited.
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

import argparse
import logging
import os
import sys

# This script is actually within our package, so add the package to the python path
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from config_system import read_config, warn_on_selected_depends

logging.basicConfig(format='%(levelname)s: %(message)s', level=logging.WARNING)

parser = argparse.ArgumentParser()
parser.add_argument('config', help='Path to the configuration file (*.config)')
parser.add_argument('-d', '--database', help='Path to the configuration database (Mconfig)', required=True)
parser.add_argument('-w', '--warning', action='store', help='Config options to warn about if selected', nargs='+', required=True)
parser.add_argument('--ignore-missing', dest="ignore_missing", action='store_true', default=False,
                    help="Ignore missing database files included with 'source'")
args = parser.parse_args()

read_config(args.database, args.config, args.ignore_missing)

warn_on_selected_depends(args.warning)
