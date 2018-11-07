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

# This script sets up the environment to be able to build the driver
# using Android paths only.

# This script should be called like:
# $PATH_TO_PROJ/generate_android_inc.bash $PATH_TO_PROJ $BUILDDIR $GOROOT

# This script is invoked by Android.mk in a $(shell) expression, so its
# standard output is buffered. Swap stdout and stderr so that the output of
# this is visible.
exec 3>&2 2>&1 1>&3-

set -e

PATH_TO_PROJ="$1"
BUILDDIR=$(readlink -f "$2")
GOROOT="$3"

if [ -x "${BUILDDIR}/buildme" -a -f "${BUILDDIR}/bob.config" ] ; then
    cd "${PATH_TO_PROJ}"

    # The following would setup to use Androids prebuilt go. We don't
    # do this because we depend on Go 1.8 but still run on Android N
    # (which has an older version).
    #export GOPATH="${ANDROID_BUILD_TOP}"
    #export GOROOT="${GOPATH}/prebuilts/go/linux-x86"
    #export PATH="${GOROOT}/bin:${PATH}"

    # Use GOROOT recorded during setup_android call
    export GOROOT

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
    echo "${BUILDDIR}/buildme and ${BUILDDIR}/bob.config don't exist!"
    exit 1
fi

echo "Success" >&2
