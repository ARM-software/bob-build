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

set -e

# Example usage
# ./bob_graph -graph_out=libMy -graph_who_uses=libMy,libOther -graph_dependencies=libMy,libOther
#
# To view dependencies of libOther
# ./bob_graph -graph_out=libOther_deps -graph_dependencies=libOther

# Switch to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# Read settings written by bootstrap.bash
source ".bob.bootstrap"

# Switch to the working directory
cd "${WORKDIR}"

BOB_BUILDER="${BUILDDIR}/.bootstrap/bin/bob"

if [ ! -f "${BOB_BUILDER}" ]; then
	echo "Please first run buildme"
	echo "Missing bob_builder: ${BOB_BUILDER}"
	exit 1
fi

echo "
#
# Legend description
#
# Nodes
# green node            - static library
# orange node           - shared library
# gray node             - binary
# yellow node           - default

# Marked node
# double circle         - marked node

# Edges
# orange edges          - content of shared_libs
# orange-dashed edges   - content of export_shared_libs
# green edges           - content of static_libs
# green-dashed edges    - content of export_static_libs
# red edges             - content of whole_static
# yellow edges          - content of defaults
#
"

"${BOB_BUILDER}" -l "${BLUEPRINT_LIST_FILE}" -b "${BUILDDIR}" "$@" "${SRCDIR}/${TOPNAME}"
