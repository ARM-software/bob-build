#!/bin/bash




set -e
trap 'echo "*** Unexpected error ***"' ERR

ORIG_PWD="$(pwd)"
export ORIG_PWD

# Move to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

source ".bob.bootstrap"

# Move to the working directory
cd -P "${WORKDIR}"

eval "${BOB_DIR}/config_system/update_config.py" -d "${SRCDIR}/Mconfig" \
    "${BOB_CONFIG_OPTS}" "${BOB_CONFIG_PLUGIN_OPTS}" \
    -j "${CONFIG_JSON}" \
    -c "${CONFIG_FILE}" \
    --depfile "${CONFIG_FILE}.d"
