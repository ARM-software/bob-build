#!/bin/bash

# Copyright 2020, 2023 Arm Limited.
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

set -e
trap 'echo "*** Unexpected error ***"' ERR

ORIG_PWD="$(pwd)"

# Move to the build directory
cd $(dirname "${BASH_SOURCE[0]}")

source ".bob.bootstrap"

# Move to the working directory
cd -P "${WORKDIR}"

ignore_missing="--ignore-missing"

if [[ ! ${BOB_CONFIG_OPTS} =~ ${ignore_missing} ]]; then
    ignore_missing=""
fi

"${BOB_DIR}/config_system/print_user_config.py" \
    -c "${CONFIG_FILE}" \
    -d "${SRCDIR}/Mconfig" \
    ${ignore_missing}
