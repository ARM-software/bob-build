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

# SRCDIR - Path to base of source tree. This can be relative to PWD or absolute.
# BUILDDIR - Build output directory. This can be relative to PWD or absolute.
# CONFIGNAME - Name of the configuration file.
# BLUEPRINT_LIST_FILE - Path to file listing all Blueprint input files.
#                       This can be relative to PWD or absolute.
# BOB_CONFIG_OPTS - Configuration options to be used when calling the
#                   configuration system.
# BOB_CONFIG_PLUGINS - Configuration system plugins to use
# TOPNAME - Name used for Bob Blueprint files.

# The location that this script is called from determines the working
# directory of the build.

set -e

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")

source "${SCRIPT_DIR}/pathtools.bash"

# Use defaults where we can. Generally the caller should set these.
if [ -z "${SRCDIR}" ] ; then
    # If not specified, assume the current directory
    export SRCDIR=.
fi

if [[ -z "$BUILDDIR" ]]; then
  echo "BUILDDIR is not set - using ."
  export BUILDDIR=.
fi

if [[ -z "$CONFIGNAME" ]]; then
  echo "CONFIGNAME is not set - using bob.config"
  export CONFIGNAME="bob.config"
fi

if [[ -z "$TOPNAME" ]]; then
  echo "TOPNAME must be set"
  exit 1
fi

if [[ -z "$BOB_CONFIG_OPTS" ]]; then
  export BOB_CONFIG_OPTS=""
fi

if [[ -z "$BOB_CONFIG_PLUGINS" ]]; then
  export BOB_CONFIG_PLUGINS=""
fi

if [ "${BUILDDIR}" = "." ] ; then
    WORKDIR=.
else
    # Create the build directory
    mkdir -p "$BUILDDIR"

    # Relative path from build directory to working directory
    WORKDIR=$(relative_path "${BUILDDIR}" $(pwd))
fi

BOOTSTRAP_GLOBFILE="${BUILDDIR}/.bootstrap/build-globs.ninja"
if [ -f "${BOOTSTRAP_GLOBFILE}" ]; then
    PREV_DIR=$(sed -n -e "s/^g.bootstrap.buildDir = \(.*\)/\1/p" "${BOOTSTRAP_GLOBFILE}")
    if [ "${PREV_DIR}" != "${BUILDDIR}" ] ; then
        # BOOTSTRAP_GLOBFILE is invalid if BUILDDIR has changed
        # Invalidate it so that the bootstrap builder can be built
        cat /dev/null > "${BOOTSTRAP_GLOBFILE}"
    fi
fi

# Calculate Bob directory relative to working directory, build directory and absolute
BOB_DIR="$(relative_path $(pwd) "${SCRIPT_DIR}")"
BOB_DIR_FROM_BUILD="$(relative_path $(bob_realpath "${BUILDDIR}") "${SCRIPT_DIR}")"
BOB_DIR_ABS="$(bob_realpath "${SCRIPT_DIR}")"

export BOOTSTRAP="${BOB_DIR}/bootstrap.bash"
export BLUEPRINTDIR="${BOB_DIR}/blueprint"

source "${BOB_DIR}/bob.bootstrap.version"

# Bootstrap blueprint.
"${BLUEPRINTDIR}/bootstrap.bash"

# Always use the host_explore config plugin
BOB_CONFIG_PLUGIN_OPTS="-p ${BOB_DIR}/scripts/host_explore"

# Add any other plugins requested by the caller
for i in $BOB_CONFIG_PLUGINS; do
    BOB_CONFIG_PLUGIN_OPTS="$BOB_CONFIG_PLUGIN_OPTS -p $i"
done

# Configure Bob in the build directory
sed -e "s|@@WorkDir@@|${WORKDIR}|" \
    -e "s|@@BuildDir@@|${BUILDDIR}|" \
    -e "s|@@SrcDir@@|${SRCDIR}|" \
    -e "s|@@BobDir@@|${BOB_DIR}|" \
    -e "s|@@PrebuiltOS@@|${PREBUILTOS}|" \
    -e "s|@@TopName@@|${TOPNAME}|" \
    -e "s|@@ListFile@@|${BLUEPRINT_LIST_FILE}|" \
    -e "s|@@ConfigName@@|${CONFIGNAME}|" \
    -e "s|@@BobConfigOpts@@|${BOB_CONFIG_OPTS}|" \
    -e "s|@@BobConfigPluginOpts@@|${BOB_CONFIG_PLUGIN_OPTS}|" \
    -e "s|@@BobBootstrapVersion@@|${BOB_VERSION}|" \
    "${BOB_DIR}/bob.bootstrap.in" > "${BUILDDIR}/.bob.bootstrap.tmp"
rsync -c "${BUILDDIR}/.bob.bootstrap.tmp" "${BUILDDIR}/.bob.bootstrap"

if [ ${SRCDIR:0:1} != '/' ]; then
    # Use relative symlinks
    ln -sf "${BOB_DIR_FROM_BUILD}/config.bash" "${BUILDDIR}/config"
    ln -sf "${BOB_DIR_FROM_BUILD}/menuconfig.bash" "${BUILDDIR}/menuconfig"
    ln -sf "${BOB_DIR_FROM_BUILD}/bob.bash" "${BUILDDIR}/bob"
    ln -sf "${BOB_DIR_FROM_BUILD}/bob_graph.bash" "${BUILDDIR}/bob_graph"
    ln -sf "${BOB_DIR_FROM_BUILD}/config_system/mconfigfmt.py" "${BUILDDIR}/mconfigfmt"
else
    # Use absolute symlinks
    ln -sf "${BOB_DIR_ABS}/config.bash" "${BUILDDIR}/config"
    ln -sf "${BOB_DIR_ABS}/menuconfig.bash" "${BUILDDIR}/menuconfig"
    ln -sf "${BOB_DIR_ABS}/bob.bash" "${BUILDDIR}/bob"
    ln -sf "${BOB_DIR_ABS}/bob_graph.bash" "${BUILDDIR}/bob_graph"
    ln -sf "${BOB_DIR_ABS}/config_system/mconfigfmt.py" "${BUILDDIR}/mconfigfmt"
fi
