#!/bin/bash

# Copyright 2018-2020 Arm Limited.
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

function bob_abspath() {
    python -c "import os, sys; print(os.path.abspath(sys.argv[1]))" "$1"
}

# Get the target of a symlink. If the target is a relative path, it is intended
# to be relative to the directory containing the link, not the current working
# directory, so it is prefixed with the link's dirname.
function bob_eval_link() {
    local link_target="$(readlink "${1}")" link_dir=
    if [[ "${link_target:0:1}" != "/" ]]; then
        link_dir="$(dirname "${1}")"
        if [[ "${link_dir: -1}" != "/" ]]; then
            link_dir="${link_dir}/"
        fi
        link_target="${link_dir}${link_target}"
    fi
    echo "${link_target}"
}

function path_is_parent() {
    local parent="$1" subpath="$2"
    if [[ ${parent} == / ]]; then
        return 0
    elif [[ ${subpath} == ${parent}/* ]]; then
        return 0
    fi
    return 1
}

# Return a path that references $2 from $1
# $1 and $2 must exist
#
# The minimum possible number of symlinks will be followed, in order to
# preserve the filesystem structure as the user sees it. Symlinks inside the
# source path *are* expanded when it is required to access their parent
# directory, because `..` on a soft link returns the parent directory of the
# link's target, rather than the link.
function relative_path() {
    [[ -e $1 ]] || { echo "relative_path: Source path '$1' does not exist" >&2; return 1; }
    [[ -e $2 ]] || { echo "relative_path: Target path '$2' does not exist" >&2; return 1; }
    local SRC_ABS=$(bob_abspath "${1}")
    local TGT_ABS=$(bob_abspath "${2}")
    local BACK= RESULT= CMN_PFX= RELPATH_FROM_LINK=

    if [[ ${TGT_ABS} == ${SRC_ABS} ]]; then
        RESULT=.

    elif path_is_parent "${SRC_ABS}" "${TGT_ABS}"; then
        # SRC_ABS is a parent of TGT_ABS

        # Remove the trailing slash from the prefix if it has one
        SRC_ABS=${SRC_ABS%/}

        RESULT=${TGT_ABS#${SRC_ABS}/}

    elif path_is_parent "${TGT_ABS}" "${SRC_ABS}"; then
        # TGT_ABS is a parent of SRC_ABS

        while [[ ${TGT_ABS} != ${SRC_ABS} ]]; do
            if [[ -L ${SRC_ABS} ]]; then
                SRC_ABS="$(bob_eval_link "${SRC_ABS}")" || return $?
                RELPATH_FROM_LINK="$(relative_path "${SRC_ABS}" "${TGT_ABS}")" || return $?
                echo "${BACK}${RELPATH_FROM_LINK}"
                return
            fi
            SRC_ABS=$(dirname ${SRC_ABS})
            BACK="../${BACK}"
        done

        RESULT=${BACK%/}

    else
        CMN_PFX=${SRC_ABS}

        while ! path_is_parent "${CMN_PFX}" "${TGT_ABS}"; do
            if [[ -L ${CMN_PFX} ]]; then
                CMN_PFX="$(bob_eval_link "${CMN_PFX}")" || return $?
                RELPATH_FROM_LINK="$(relative_path "${CMN_PFX}" "${TGT_ABS}")" || return $?
                echo "${BACK}${RELPATH_FROM_LINK}"
                return
            fi
            CMN_PFX=$(dirname ${CMN_PFX})
            BACK="../${BACK}"
        done

        # Remove the trailing slash from the prefix if it has one
        CMN_PFX=${CMN_PFX%/}

        RESULT=${BACK}${TGT_ABS#${CMN_PFX}/}
    fi

    echo ${RESULT}
}
