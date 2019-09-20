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

# Switch to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# Read settings written by bootstrap.bash
source ".bob.bootstrap"

# Switch to the working directory
cd "${WORKDIR}"

# Get Bob bootstrap version
source "${BOB_DIR}/bob.bootstrap.version"

if [[ "${BOB_BOOTSTRAP_VERSION}" != "${BOB_VERSION}" ]]; then
    echo "This build directory must be re-bootstrapped. Bob has changed since this output directory was bootstrapped." >&2
    exit 1
fi

# Refresh the configuration. This means that options changed or added since the
# last build will be chosen from their defaults automatically, so that users
# don't have to reconfigure manually if the config database changes.
python "${BOB_DIR}/config_system/generate_config_json.py" \
       "${BUILDDIR}/${CONFIGNAME}" --database "${SRCDIR}/Mconfig" \
       --json "${BUILDDIR}/config.json" ${BOB_CONFIG_OPTS}

# Get a hash of the environment so we can detect if we need to
# regenerate the build.ninja
python "${BOB_DIR}/scripts/env_hash.py" "${BUILDDIR}/.env.hash"

# Source the pathtools script - we need bob_realpath for CCACHE_BASEDIR.
source "${BOB_DIR}/pathtools.bash"

# If enabled, the following environment variables optimize the performance
# of ccache. Otherwise they have no effect.
# To build with ccache, set the environment variable CCACHE_DIR to where the
# cache is to reside and add CCACHE=y to the build config to enable.
export CCACHE_BASEDIR="$(bob_realpath ${SRCDIR})"
export CCACHE_CPP2=y
export CCACHE_SLOPPINESS=file_macro,time_macros

# Build the builder if necessary
BUILDDIR="${BUILDDIR}" SKIP_NINJA=true ${BOB_DIR}/blueprint/blueprint.bash

# Do the actual build
"${NINJA}" -f "${BUILDDIR}/build.ninja" -w dupbuild=err "$@"
