#!/usr/bin/env bash

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

# This script sets up the source tree to use Bob Blueprint under Android.

# Copy the Blueprint version of the Android makefile into place, bootstrap
# Bob with a BUILDDIR in the Android output directory, and generate an initial
# config based on the args passed to this script.

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
BOB_DIR=bob-build
PROJ_NAME="bob_example"

BASENAME=$(basename $0)
function usage {
    cat <<EOF
$BASENAME

Sets up the Bob to build for Android.

Usage:
 $BASENAME CONFIG_OPTIONS...
 $BASENAME --menuconfig

  CONFIG_OPTIONS is a list of configuration items that can include .config
  profiles and explicit options (like DEBUG=y)

Options
  -m, --menuconfig  Open graphical configuration editor
  -h, --help        Help text

Examples:
  $BASENAME ANDROID_N=y DEBUG=n
  $BASENAME --menuconfig
EOF
}

MENU=0
PARAMS=$(getopt -o hm -l help,menuconfig --name $(basename "$0") -- "$@")

eval set -- "$PARAMS"
unset PARAMS

while true ; do
    case $1 in
        -m | --menuconfig)
            MENU=1
            shift
            ;;
        --)
            shift
            break
            ;;
        -h | --help | *)
            usage
            exit 1
            ;;
    esac
done

[[ -n ${OUT} ]] || { echo "\$OUT is not set - did you run 'lunch'?"; exit 1; }
[[ -n ${ANDROID_BUILD_TOP} ]] || { echo "\$ANDROID_BUILD_TOP is not set - did you run 'lunch'?"; exit 1; }

source "${SCRIPT_DIR}/${BOB_DIR}/pathtools.bash"

PROJ_DIR=$(relative_path "${ANDROID_BUILD_TOP}" "${SCRIPT_DIR}")

# Change to the working directory
cd "${ANDROID_BUILD_TOP}"

### Variables required for Bob and Android.mk bootstrap ###

# This must match the path derived from LOCAL_MODULE and LOCAL_MODULE_CLASS
# in Android.mk.blueprint.
ANDROIDMK_DIR="${OUT}/gen/STATIC_LIBRARIES/${PROJ_NAME}_intermediates"
export BUILDDIR="${ANDROIDMK_DIR}"
export TOPNAME="build.bp"
export CONFIGNAME="bob.config"
export SRCDIR="${PROJ_DIR}"
export BLUEPRINT_LIST_FILE="${SRCDIR}/bplist"

# Write the Android.mk
TMP_ANDROID_MK="$(mktemp)"
sed -e "s#@@BOB_PROJ_NAME@@#$PROJ_NAME#" \
    -e "s#@@BOB_DIR@@#$BOB_DIR#" \
    -e "s#@@CONFIGNAME@@#$CONFIGNAME#" \
    "${PROJ_DIR}/Android.mk.blueprint" > "$TMP_ANDROID_MK"
rsync --checksum "$TMP_ANDROID_MK" "${PROJ_DIR}/Android.mk"
rm -f "$TMP_ANDROID_MK"

# Bootstrap Bob
"${PROJ_DIR}/${BOB_DIR}/bootstrap_androidmk.bash"


# Pick up some info that bob has worked out
BOOTSTRAP=".bob.bootstrap"
source "${BUILDDIR}/${BOOTSTRAP}"

# Setup the buildme script to just run bob
ln -sf "bob" "${BUILDDIR}/buildme"

if [ ! -z "$*" ] || [ ! -f "$ANDROIDMK_DIR/$CONFIGNAME" ] ; then
    # Have arguments or missing bob.config. Run config.
    "$ANDROIDMK_DIR/config" ANDROID=y "$@"
fi

if [ $MENU -eq 1 ] ; then
    "$ANDROIDMK_DIR/menuconfig"
fi
