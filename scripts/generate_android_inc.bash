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

# This script should be called like:
# $PATH_TO_PROJ/generate_android_inc.bash $PATH_TO_PROJ $BUILDDIR $GOROOT $CONFIGNAME

# This script is invoked by Android.mk in a $(shell) expression, so its
# standard output is buffered. Swap stdout and stderr so that the output of
# this is visible.
exec 3>&2 2>&1 1>&3-

set -e
trap 'echo "*** Unexpected error in $0 ***"' ERR

BOB_DIR=$(dirname $(dirname "${BASH_SOURCE[0]}"))
PATH_TO_PROJ="$1"
BUILDDIR=$(readlink -f "$2")
CONFIGNAME="$3"

source "${BOB_DIR}/pathtools.bash"

if [ -x "${BUILDDIR}/buildme" -a -f "${BUILDDIR}/${CONFIGNAME}" ] ; then
    # The Ninja path is relative to the root of the Android tree, but Bob is
    # run from the project directory.
    NINJA=`relative_path "${PATH_TO_PROJ}" "${NINJA}"`

    cd "${PATH_TO_PROJ}"

    # Use the Go shipped with Android on P and later, where it's recent enough (> 1.9).
    [[ $PLATFORM_SDK_VERSION -ge 28 ]] && export GOROOT="${TOP}/prebuilts/go/linux-x86/"

    "$BUILDDIR/buildme"

    # To integrate with Kati, it is important that we output a different
    # value in the shell whenever the generated Andoid makefiles (*.inc)
    # change.
    #
    # To do that, call md5sum is output after the build of the Android file
    # has finished. This script would normally not be called manually, but
    # rather through Android.mk
    md5sum "$BUILDDIR"/*.inc | md5sum - >&2
else
    echo "${BUILDDIR}/buildme and ${BUILDDIR}/${CONFIGNAME} don't exist!"
    exit 1
fi

echo "Success" >&2
