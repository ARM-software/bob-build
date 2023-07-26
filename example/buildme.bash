#!/bin/bash




set -e

# Check the user isn't trying to execute this script directly
if [[ ! -L "${BASH_SOURCE[0]}" ]] ; then
    echo "$0 must be executed from the build directory created by bootstrap_linux.bash"
    exit 1
fi

# Switch to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

source ".bob.bootstrap"

# Move to the working directory
cd "${WORKDIR}"

# Check for missing configuration
if [ ! -f "${CONFIG_FILE}" ] ; then
    echo "${CONFIG_FILE} is missing. Use config or menuconfig to configure the project."
    exit 1
fi

# Build
"${BUILDDIR}/bob" "$@"
