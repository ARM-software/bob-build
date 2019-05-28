#!/usr/bin/env bash

# Copyright 2019 Arm Limited.
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

set -eu
trap 'echo "*** Unexpected error in $0 ***"' ERR

BOB_DIR=$(dirname "${BASH_SOURCE[0]}")

function die {
    echo "${BASH_SOURCE[0]}: ${*}"
    exit 1
}

# ${VAR:-} will substitute an empty string if the variable is unset, which
# stops `set -u` complaining before `die` is invoked.
[[ -z ${SRCDIR:-} ]] && die "\$SRCDIR not set"
[[ -e ${SRCDIR}/Android.mk ]] && die "${SRCDIR}/Android.mk conflicts with Android.bp. Please remove!"

# Set up Bob's Android.bp
pushd "${BOB_DIR}" >&/dev/null
ln -fs Android.bp.in Android.bp
popd >&/dev/null

# Create an `Android.bp` symlink for every `build.bp` present in the source
# dir, and remove dead links.
pushd "${SRCDIR}" >&/dev/null
find -name build.bp -execdir ln -fs build.bp Android.bp \;
find -name Android.bp -xtype l -delete
popd >&/dev/null
