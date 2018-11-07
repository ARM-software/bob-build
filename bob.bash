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

# Switch to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# Read settings written by bootstrap.bash
BOOTSTRAP=".bob.bootstrap"
source "${BOOTSTRAP}"

# Switch to the working directory
cd "${WORKDIR}"

# Generate config.go from Mconfig
python "${BOB_DIR}/scripts/generate_config_json.py" --database "${SRCDIR}/Mconfig" --output "${BUILDDIR}/config.json" ${BOB_CONFIG_OPTS} "${BUILDDIR}/${CONFIGNAME}"

# If enabled, the following environment variables optimize the performance
# of ccache. Otherwise they have no effect.
# To build with ccache, set the environment variable CCACHE_DIR to where the
# cache is to reside and add CCACHE=y to the build config to enable.
export CCACHE_BASEDIR="$(readlink -f "${SRCDIR}")"
export CCACHE_CPP2=y
export CCACHE_SLOPPINESS=file_macro,time_macros

# Build the builder if necessary
BUILDDIR="${BUILDDIR}" SKIP_NINJA=true ${BOB_DIR}/blueprint/blueprint.bash

# Do the actual build
ninja -f "${BUILDDIR}/build.ninja" -w dupbuild=err "$@"
