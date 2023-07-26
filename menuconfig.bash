#!/bin/bash




set -e
trap 'echo "*** Unexpected error ***"' ERR

# Move to the build driectory
cd "$(dirname "${BASH_SOURCE[0]}")"
source ".bob.bootstrap"

# Move to the working directory
cd -P "${WORKDIR}"

eval "${BOB_DIR}/config_system/menuconfig.py" -d "${SRCDIR}/Mconfig" \
    "${BOB_CONFIG_OPTS}" "${BOB_CONFIG_PLUGIN_OPTS}" \
    -j "${CONFIG_JSON}" \
    --depfile "${CONFIG_FILE}.d" \
    "${CONFIG_FILE}"
