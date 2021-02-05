#!/bin/bash

# Copyright 2018-2021 Arm Limited.
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
# CONFIGDIR - Configuration directory. This can be relative to PWD or absolute.
# CONFIGNAME - Name of the configuration file.
# BLUEPRINT_LIST_FILE - Path to file listing all Blueprint input files.
#                       This can be relative to PWD or absolute.
# BOB_CONFIG_OPTS - Configuration options to be used when calling the
#                   configuration system.
# BOB_CONFIG_PLUGINS - Configuration system plugins to use

# The location that this script is called from determines the working
# directory of the build.

set -e

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")

source "${SCRIPT_DIR}/pathtools.bash"
source "${SCRIPT_DIR}/bootstrap/utils.bash"

# Use defaults where we can. Generally the caller should set these.
if [ -z "${SRCDIR}" ] ; then
    # If not specified, assume the current directory
    export SRCDIR=.
fi

if [[ -z "$BUILDDIR" ]]; then
  echo "BUILDDIR is not set - using ."
  export BUILDDIR=.
fi

if [[ -z "$CONFIGDIR" ]]; then
  CONFIGDIR="${BUILDDIR}"
else
  mkdir -p "${CONFIGDIR}"
fi

if [[ -z "$CONFIGNAME" ]]; then
  echo "CONFIGNAME is not set - using bob.config"
  CONFIGNAME="bob.config"
fi

if [[ -z "$BOB_CONFIG_OPTS" ]]; then
  BOB_CONFIG_OPTS=""
fi

if [[ -z "$BOB_CONFIG_PLUGINS" ]]; then
  BOB_CONFIG_PLUGINS=""
fi

# FIXME: It was found that using an absolute path for the build directory (inside srcdir)
# could lead to dependency issues later, in part due to ninja not being able to handle well
# absolute depfile paths (see https://github.com/ninja-build/ninja/issues/1251).
# This issue only happens if the build directory is located in the source directory.
# For now, let's warn users about this potential issue.
if [ "${BUILDDIR:0:1}" = '/' ] && $(path_is_parent $(bob_abspath "${SRCDIR}") "${BUILDDIR}"); then
  echo "WARNING! Using an absolute path for the build directory located inside the source directory" \
" can lead to dependency issues. Recommend using a relative path for the build directory."
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
        # On OSX, also force a rebuild of microfactory
        if [ "$(uname)" = "Darwin" ] ; then
            rm -f "${BUILDDIR}/.minibootstrap/microfactory_$(uname)"
        fi
    fi
fi

# Calculate Bob directory relative to the working directory.
BOB_DIR="$(relative_path $(pwd) "${SCRIPT_DIR}")"
CONFIG_FILE="${CONFIGDIR}/${CONFIGNAME}"
CONFIG_JSON="${CONFIGDIR}/.bob.config.json"

export TOPNAME="build.bp"
export BOOTSTRAP="${BOB_DIR}/bootstrap.bash"
export BLUEPRINTDIR="${BOB_DIR}/blueprint"

# Bootstrap blueprint.
"${BLUEPRINTDIR}/bootstrap.bash"

# Configure Bob in the build directory
write_bootstrap

if [ ${SRCDIR:0:1} != '/' ]; then
    # Use relative symlinks
    BOB_DIR_FROM_BUILD="$(relative_path $(bob_realpath "${BUILDDIR}") "${SCRIPT_DIR}")"
else
    # Use absolute symlinks
    BOB_DIR_FROM_BUILD="$(bob_realpath "${SCRIPT_DIR}")"
fi
create_config_symlinks "${BOB_DIR_FROM_BUILD}" "${BUILDDIR}"
create_bob_symlinks "${BOB_DIR_FROM_BUILD}" "${BUILDDIR}"
