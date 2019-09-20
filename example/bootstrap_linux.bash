#!/bin/bash
#
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

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
BOB_DIR=bob-build

# Currently SRCDIR must be absolute. Because it's derived from
# SCRIPT_DIR, make SCRIPT_DIR absolute so that we get SRCDIR for free.
if [ "${SCRIPT_DIR:0:1}" != '/' ]; then
    SCRIPT_DIR="$(readlink -f "${SCRIPT_DIR}")"
fi

# Select a BUILDDIR if not provided one
if [[ -z "$BUILDDIR" ]]; then
    echo "BUILDDIR is not set - using build"
    BUILDDIR=build
fi

# Create the build directory. The 'relative_path' helper can only be used on
# existing directories.
mkdir -p "$BUILDDIR"

source "${SCRIPT_DIR}/${BOB_DIR}/pathtools.bash"

ORIG_BUILDDIR="${BUILDDIR}"

SRCDIR="${SCRIPT_DIR}"
if [ "${BUILDDIR:0:1}" != '/' ]; then
    # Redo BUILDDIR to be relative to SRCDIR
    BUILDDIR="$(relative_path "${SRCDIR}" "${BUILDDIR}")"
fi

# Move to the source directory - we want this to be the working directory of the build
cd "${SRCDIR}"

# Set variables to bootstrap Bob.
export TOPNAME="build.bp"
export BLUEPRINT_LIST_FILE="bplist"
export CONFIGNAME="bob.config"
export BOB_CONFIG_OPTS=
export BOB_CONFIG_PLUGINS=
export SRCDIR
export BUILDDIR

"${BOB_DIR}/bootstrap_linux.bash"

# Pick up some info that bob has worked out
source "${BUILDDIR}/.bob.bootstrap"

# Setup the build script
if [ "${SRCDIR:0:1}" != '/' ]; then
    # Use a relative symlink
    if [ "${SRCDIR}" != '.' ]; then
        ln -sf "${WORKDIR}/${SRCDIR}/buildme.bash" "${BUILDDIR}/buildme"
    else
        ln -sf "${WORKDIR}/buildme.bash" "${BUILDDIR}/buildme"
    fi
else
    # Use an absolute symlink
    ln -sf "${SRCDIR}/buildme.bash" "${BUILDDIR}/buildme"
fi

# Print info for users
echo "To configure the build directory, run ${ORIG_BUILDDIR}/config ARGS"
echo "Then build with ${ORIG_BUILDDIR}/buildme"
