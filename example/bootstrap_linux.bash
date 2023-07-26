#!/bin/bash
#



SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
BOB_DIR=bob-build

source "${SCRIPT_DIR}/${BOB_DIR}/pathtools.bash"

# Select a BUILDDIR if not provided one
if [[ -z "$BUILDDIR" ]]; then
    echo "BUILDDIR is not set - using 'build'"
    BUILDDIR=build
fi

# Create the build directory. The 'relative_path' helper can only be used on
# existing directories.
mkdir -p "$BUILDDIR"

# Currently SRCDIR must be absolute.
SRCDIR="$(bob_abspath ${SCRIPT_DIR})"

ORIG_BUILDDIR="${BUILDDIR}"
if [ "${BUILDDIR:0:1}" != '/' ]; then
    # Redo BUILDDIR to be relative to SRCDIR
    BUILDDIR="$(relative_path "${SRCDIR}" "${BUILDDIR}")"
fi

# Move to the source directory - we want this to be the working directory of the build
cd "${SRCDIR}" || exit

# Export data needed for Bob bootstrap script
export SRCDIR
export BUILDDIR
export CONFIGNAME="bob.config"
export BOB_CONFIG_OPTS=
export BOB_CONFIG_PLUGINS=
export BLUEPRINT_LIST_FILE="bplist"

# Bootstrap Bob (and Blueprint)
"${BOB_DIR}/bootstrap_linux.bash"

# Pick up some info that bob has worked out
source "${BUILDDIR}/.bob.bootstrap"

# Setup the buildme script
if [ "${SRCDIR:0:1}" != '/' ]; then
    # Use a relative symlink
    if [ "${SRCDIR}" != '.' ]; then
        ln -sf "${WORKDIR}/${SRCDIR}/buildme.bash" "${BUILDDIR}/buildme"
    else
        ln -sf "${WORKDIR}/buildme.bash" "${BUILDDIR}/buildme"
    fi
else
    # Use an absolute symlink
    ln -sf "${SRCDIR}/buildme.bash" "${BUILDDIR}/buildme"
fi

# Print info for users
echo "To configure the build directory, run ${ORIG_BUILDDIR}/config ARGS"
echo "Then build with ${ORIG_BUILDDIR}/buildme"
