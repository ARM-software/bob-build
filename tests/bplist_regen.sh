#!/bin/sh

# Copyright 2018, 2020 Arm Limited.
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

# Create a list of build.bp files

set -e

SRCDIR="$(dirname "${0}")"

LIST_FILE=bplist
TEMP_LIST_FILE="${LIST_FILE}.tmp"

cd "${SRCDIR}"

# Locate all build.bp under the current directory. Exclusions:
# * hidden directories (starting with .)
# * Bob build directories (these contain a file .out-dir)
find . -mindepth 1 \
     -type d \( -name ".*" -o -execdir test -e {}/.out-dir \; \) -prune \
     -o -name build.bp -print > "${TEMP_LIST_FILE}"

echo ./bob/Blueprints >> "${TEMP_LIST_FILE}"
echo ./bob/blueprint/Blueprints >> "${TEMP_LIST_FILE}"

LC_ALL=C sort "${TEMP_LIST_FILE}" -o "${TEMP_LIST_FILE}"

if cmp -s "${LIST_FILE}" "${TEMP_LIST_FILE}"; then
  rm "${TEMP_LIST_FILE}"
else
  mv "${TEMP_LIST_FILE}" "${LIST_FILE}"
fi
