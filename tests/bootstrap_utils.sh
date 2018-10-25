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

function create_link() {
    local target="$1" linkname="$2"

    if [[ ! -e "$linkname" ]]; then
        # Link doesn't exist
        :
    elif [[ -L "$linkname" ]]; then
        # Link exists. Check if it already points to the right place.
        if [[ $(readlink "$linkname") = "$target" ]]; then
            return 0 # Already refers to the correct location
        else
            rm "$linkname" || return $?
        fi
    else
        # Link location is already in use - fail rather than deleting it
        # without warning.
        echo "Can't create link from $linkname to $target - $linkname already exists and is not a symbolic link"
        return 1
    fi
    ln -s "$target" "$linkname"
}
