#!/bin/bash

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

set -e
trap 'echo "*** Unexpected error ***"' ERR

# Move to the build driectory
cd $(dirname "${BASH_SOURCE[0]}")
BOOTSTRAP=".bob.bootstrap"
source "${BOOTSTRAP}"

# Move to the working directory
cd "${WORKDIR}"
exit_status=0

"${BOB_DIR}/config_system/menuconfig.py" -d "${SRCDIR}/Mconfig" ${BOB_CONFIG_OPTS} ${BOB_CONFIG_PLUGIN_OPTS} -o "${BUILDDIR}/${CONFIGNAME}.tmp" "${BUILDDIR}/${CONFIGNAME}" || exit_status=$?

# when warnings/errors occurred we still want to sync settings
rsync -I -c "${BUILDDIR}/${CONFIGNAME}.tmp" "${BUILDDIR}/${CONFIGNAME}"

if [ "$exit_status" -eq "1" ]; then # warnings occurred
    exit 0
else
    exit $exit_status
fi
