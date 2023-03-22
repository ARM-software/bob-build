#!/bin/bash

# Copyright 2018-2019, 2023 Arm Limited.
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

# Mounts the components of Bob into a directory structure that Go
# tools and editors expect

BASENAME=$(basename $0)

function usage() {
    cat <<EOF
$BASENAME

Sets up a directory with Bob components that will work with Go tools.
Requires bindfs.

If a path isn't specified, GOPATH will be consulted.

Usage:
 $BASENAME [path]
 $BASENAME -u path

Options
  -u    Undo
  -h    Help text
EOF
}

# shellcheck disable=SC2068,SC2294

function run() {
    echo $@
    eval $@
}

function bind() {
    mkdir -p "$2"
    run bindfs --no-allow-other "$1" "$2"
}

function unbind() {
    run fusermount -u $1
}

BOB_PATH="$(dirname $0)/.."

PARAMS=$(getopt -o uh --name ${BASENAME} -- "$@")

eval set -- "$PARAMS"
unset PARAMS

UNBIND=0
while true ; do
    case $1 in
        -u)
            UNBIND=1
            shift
            ;;
        --)
            shift
            break
            ;;
        -h | *)
            usage
            exit 1
            ;;
    esac
done

OUTPUT_PATH="$1"


if [ -z "${OUTPUT_PATH}" ] ; then
    # Try to get workspace from GOPATH
    # if GOPATH contains multiple paths, use the first one
    OUTPUT_PATH="${GOPATH%%:*}"
fi

if [ -z "${OUTPUT_PATH}" ]; then
    echo "Error: could not determine where to create a Go-compatible workspace for Bob. Please set GOPATH or specify the desired destination directory as an argument"
    exit 1
fi

if [ $UNBIND -eq 0 ]; then
    bind "${BOB_PATH}/blueprint" "${OUTPUT_PATH}/src/github.com/google/blueprint"
    bind "${BOB_PATH}" "${OUTPUT_PATH}/src/github.com/ARM-software/bob-build"

    GOPATH=${OUTPUT_PATH} go get github.com/stretchr/testify

    echo "Go-compatible workspace created at ${OUTPUT_PATH}"
else
    unbind "${OUTPUT_PATH}/src/github.com/google/blueprint"
    unbind "${OUTPUT_PATH}/src/github.com/ARM-software/bob-build"

    echo "Mounts unbound"
fi
