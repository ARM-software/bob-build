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

# Check that relative_path() works as expected
SCRIPT_DIR=$(dirname $0)
BOB_DIR="${SCRIPT_DIR}/.."

source "${BOB_DIR}/pathtools.bash"

HAVE_FAILURE=0

function test_relpath() {
    local SOURCE="${1}"
    local TARGET="${2}"
    local EXPECTED="${3}"

    local RESULT=$(relative_path "${SOURCE}" "${TARGET}")

    if [ "${RESULT}" != "${EXPECTED}" ] ; then
        echo FAIL: relative_path ${SOURCE} ${TARGET} expected to return ${EXPECTED}, got ${RESULT}
        HAVE_FAILURE=1
    fi
}

if [ -e "a" ] ; then
    echo "Abort: Need to create temporary directory heirarchy to test. 'a' already exists"
    exit 1
fi
if [ -e "x" ] ; then
    echo "Abort: Need to create temporary directory heirarchy to test. 'x' already exists"
    exit 1
fi

mkdir -p "a/b/c"
mkdir -p "a/b2/c"
mkdir -p "a/b/g"
mkdir -p "a/e/g"
mkdir -p "x/y/z"

# Directory is the same
test_relpath "a"     "a"     "."
test_relpath "a/b"   "a/b"   "."
test_relpath "a/b/c" "a/b/c" "."

# Target is a subdirectory
test_relpath "a"   "a/b"   "b"
test_relpath "a"   "a/b/c" "b/c"
test_relpath "a/b" "a/b/c" "c"
test_relpath "a"   "a/e"   "e"
test_relpath "a"   "a/e/g" "e/g"
test_relpath "a/e" "a/e/g" "g"

# Target is a parent
test_relpath "a/b/c" "a"   "../.."
test_relpath "a/b/c" "a/b" ".."
test_relpath "a/e/g" "a"   "../.."
test_relpath "a/e/g" "a/e" ".."

# Target shares a common parent
test_relpath "a/b"   "a/e"   "../e"
test_relpath "a/b/c" "a/e/g" "../../e/g"
test_relpath "a/b/c" "a/b/g" "../g"

# No shared path (actually this shares the current dir)
test_relpath "a/b/c" "x/y/z" "../../../x/y/z"

# Check directory substring mismatches
test_relpath "a/b" "a/b2/c"  "../b2/c"
test_relpath "a/b2/c" "a/b"  "../../b"

# Check the special case where the common root is `/`
test_relpath "/usr" "/bin" "../bin"
test_relpath "/usr/local/bin" "/bin/bash" "../../../bin/bash"
test_relpath "/" "/usr/include/stdio.h" "usr/include/stdio.h"
test_relpath "/bin" "/" ".."

# Cleanup
rm -rf a x

exit ${HAVE_FAILURE}
