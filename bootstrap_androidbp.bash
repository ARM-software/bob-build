#!/bin/bash

# Copyright 2020 Arm Limited.
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

source $(dirname "${BASH_SOURCE[0]}")/bootstrap.bash

BOB_DIR=$(dirname "${BASH_SOURCE[0]}")

function die {
    echo "${BASH_SOURCE[0]}: ${*}"
    exit 1
}

# ${VAR:-} will substitute an empty string if the variable is unset, which
# stops `set -u` complaining before `die` is invoked.
[[ -z ${SRCDIR:-} ]] && die "\$SRCDIR not set"
[[ -z ${PROJ_NAME:-} ]] && die "\$PROJ_NAME not set"

# Remove any stalled symlinks from Soong plugin bootstrap
pushd "${SRCDIR}" >&/dev/null
find -name Android.bp -type l -delete
popd >&/dev/null

# Set up Android.bp with plugins
TMP_ANDROID_BP=$(mktemp)
sed -e "s#@@PROJ_NAME@@#${PROJ_NAME}#" \
    "${BOB_DIR}/plugins/Android.bp.in" > "${TMP_ANDROID_BP}"
rsync --checksum "${TMP_ANDROID_BP}" "${BOB_DIR}/plugins/Android.bp"
rm -f "${TMP_ANDROID_BP}"
