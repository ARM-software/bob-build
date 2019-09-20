#!/bin/bash

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

set -e

# Check the user isn't trying to execute this script directly
if [[ ! -L "${BASH_SOURCE[0]}" ]] ; then
    echo "$0 must be executed from the build directory created by bootstrap_linux.bash"
    exit 1
fi

# Switch to the build directory
cd $(dirname "${BASH_SOURCE[0]}")

source ".bob.bootstrap"

# Check for missing configuration
if [ ! -f "${CONFIGNAME}" ] ; then
    echo "${CONFIGNAME} is missing. Use config or menuconfig to configure the project."
    exit 1
fi

# Move to the working directory
cd "${WORKDIR}"

# Build
"${BUILDDIR}/bob" "$@"
