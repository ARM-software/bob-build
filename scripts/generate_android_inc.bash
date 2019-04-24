#!/usr/bin/env bash

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

# This script sets up the environment to be able to build the project
# using Android paths only.

# This script is invoked by Android.mk in a $(shell) expression, so its
# standard output is buffered. Create a new file descriptor, 9, which refers to
# stdout, and is used to control the buffered output that Android.mk sees.
# Redirect all other output (stdout *and* stderr) from this script to stderr,
# so that it is seen by the user when running `make`.
exec 9>&1 1>&2

set -e
trap 'echo "*** Unexpected error in $0 ***"' ERR

BOB_DIR=$(dirname $(dirname "${BASH_SOURCE[0]}"))

while getopts "c:o:s:v:" opt; do
    case $opt in
        c) CONFIGNAME="$OPTARG";;
        o) BUILDDIR=`readlink -f "$OPTARG"`;;
        s) PATH_TO_PROJ="$OPTARG";;
        v) PLATFORM_SDK_VERSION="$OPTARG";;
    esac
done

if [[ -z $BUILDDIR || -z $CONFIGNAME || -z $PATH_TO_PROJ || -z $PLATFORM_SDK_VERSION ]]; then
    echo "Error: Missing argument to $0"
    echo "Usage: $0 -c CONFIGNAME -o BUILDDIR -s PATH_TO_PROJ -v PLATFORM_SDK_VERSION"
    exit 1
fi

source "${BOB_DIR}/pathtools.bash"

if ! [[ -x ${BUILDDIR}/buildme ]]; then
    echo "${BUILDDIR}/buildme does not exist!"
    exit 1
elif ! [[ -f "${BUILDDIR}/${CONFIGNAME}" ]]; then
    echo "${BUILDDIR}/${CONFIGNAME} does not exist!"
    exit 1
elif [[ -z $NINJA ]]; then
    echo "\$NINJA not set!"
    exit 1
else
    # Use the Go shipped with Android on P and later, where it's recent enough (>= 1.10).
    [[ ${PLATFORM_SDK_VERSION} -ge 28 ]] && export GOROOT=prebuilts/go/linux-x86/

    "$BUILDDIR/buildme"

    # To integrate with Kati, it is important that we output a different
    # value in the shell whenever the generated Andoid makefiles (*.inc)
    # change.
    #
    # To do that, call md5sum is output after the build of the Android file
    # has finished. This script would normally not be called manually, but
    # rather through Android.mk
    md5sum "$BUILDDIR"/*.inc | md5sum - >&9
fi

echo "Success" >&9
