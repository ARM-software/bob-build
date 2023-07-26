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

ignore_missing="--ignore-missing"

if [[ ! ${BOB_CONFIG_OPTS} =~ ${ignore_missing} ]]; then
    ignore_missing=""
fi

"${BOB_DIR}/config_system/print_user_config.py" \
    -c "${CONFIG_FILE}" \
    -d "${SRCDIR}/Mconfig" \
    "${ignore_missing}"
