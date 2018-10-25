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

# Functions for manipulating paths

# Counts the number of path elements
function count_path_elems() {
    P=$1
    IFS='/'
    set -f $P
    echo $#
}

# Choose the shortest path of the 2 arguments.
# This is expected to be used with equivalent relative and absolute paths.
# Where they are the same length, the first is preferred.
function shortest_path() {
    COUNT1=$(count_path_elems $1)
    COUNT2=$(count_path_elems $2)
    if [ ${COUNT1} -le ${COUNT2} ]; then
        echo ${1}
    else
        echo ${2}
    fi
}

# Portable version of readlink. There are no requirements on path components existing.
if which realpath >&/dev/null &&
   [[ -n "$(realpath --version 2>&1 | grep 'GNU coreutils')" ]]; then
    function bob_realpath {
        realpath -m "$1"
    }
else
    function bob_realpath() {
        python -c "import os, sys; print(os.path.realpath(sys.argv[1]))" "$1"
    }
fi

# Return a path that references $2 from $1
# $1 and $2 must exist
# This is a simple implementation. We rely on readlink to sort out symlink issues for us.
# If there are fewer path elements in the absolute version, return that instead.
function relative_path() {
    SRC_ABS=$(bob_realpath $1)
    TGT_ABS=$(bob_realpath $2)

    BACK=
    # ${TGT_ABS#SRC_ABS} removes the SRC_ABS from TGT_ABS
    if [ "${TGT_ABS#${SRC_ABS}}" != "${TGT_ABS}" ]; then
        # SRC_ABS is a parent of TGT_ABS

        RESULT=${TGT_ABS#${SRC_ABS}/}

    elif [ "${SRC_ABS#${TGT_ABS}}" != "${SRC_ABS}" ]; then
        # TGT_ABS is a parent of SRC_ABS

        while [ "${TGT_ABS}" != "${SRC_ABS}" ]; do
            SRC_ABS=$(dirname ${SRC_ABS})
            BACK="../${BACK}"
        done

        RESULT=${BACK%/}

    else
        CMN_PFX=${SRC_ABS}

        # ${TGT_ABS#CMN_PFX} removes the CMN_PFX from TGT_ABS
        # When we have the correct prefix, the condition will fail
        while [ "${TGT_ABS#${CMN_PFX}}" = "${TGT_ABS}" ]; do
            CMN_PFX=$(dirname ${CMN_PFX})
            BACK="../${BACK}"
        done

        RESULT=${BACK}${TGT_ABS#${CMN_PFX}/}
    fi

    echo ${RESULT}
}
