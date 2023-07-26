#!/bin/bash




set -e

# Example usage
# ./bob_graph --graph-start-nodes=libMy,libOther
#
# To view users of libOther
# ./bob_graph --graph-start-nodes=libOther --graph-rev-deps

# Switch to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# Read settings written by bootstrap.bash
source ".bob.bootstrap"

# Switch to the working directory
cd -P "${WORKDIR}"

BOB_BUILDER_TARGET=".bootstrap/bin/bob"
BOB_BUILDER="${BUILDDIR}/${BOB_BUILDER_TARGET}"
BOB_BUILDER_NINJA="${BUILDDIR}/.bootstrap/build.ninja"

if [ ! -f "${BOB_BUILDER_NINJA}" ]; then
	echo "Missing ${BOB_BUILDER_NINJA}"
	echo "Please build your project first"
	exit 1
fi

# Make sure Bob is built
ninja -f "${BOB_BUILDER_NINJA}" "${BOB_BUILDER_TARGET}"

echo "
#
# Legend
#
# Nodes
# green           - static library
# orange          - shared library
# gray            - binary
# blue            - ldlib flag
# yellow          - defaults module
# white           - external library (not defined in Bob)

# Marked node
# double circle   - marked node

# Edges
# orange edge     - linked by shared_libs
# green edge      - linked by static_libs
# red edge        - linked by whole_static
# blue edge       - linked by ldlibs
# yellow edge     - uses defaults
# dashed edge     - dependency propagated to closest binary or shared library
"

"${BOB_BUILDER}" -l "${BLUEPRINT_LIST_FILE}" -b "${BUILDDIR}" "$@" "${SRCDIR}/${TOPNAME}"
