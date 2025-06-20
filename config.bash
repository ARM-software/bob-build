#!/bin/bash




set -e
trap 'echo "*** Unexpected error ***"' ERR

ORIG_PWD="$(pwd)"

# Move to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")" || exit
source ".bob.bootstrap"

declare -a ARG_TARGET

for arg in "$@"
do

    if [[ $arg =~ "=" ]];then
        ARG_TARGET+=(\'"${arg}"\')
    elif [ "${arg:0:1}" == "/" ];then
        ARG_TARGET+=("${arg}")
    else
        if [ -f "${ORIG_PWD}/${arg}" ];then
            ARG_TARGET+=("${ORIG_PWD}/${arg}")
        else
            ARG_TARGET+=("${SRCDIR}/bldsys/profiles/${arg}")
        fi
    fi
done

# Move to the working directory
cd -P "${WORKDIR}"

# Allow passed in `update_config`
if command -v bazel-bob-update-config.exe &> /dev/null
then
  UPDATE_CONFIG=bazel-bob-update-config.exe
else
  UPDATE_CONFIG="${BOB_DIR}/config_system/update_config.py"
fi

# shellcheck disable=SC2294
eval "${UPDATE_CONFIG}" --new -d "${SRCDIR}/Mconfig" \
    "${BOB_CONFIG_OPTS}" "${BOB_CONFIG_PLUGIN_OPTS}" \
    -j "${CONFIG_JSON}" \
    -c "${CONFIG_FILE}" \
    --depfile "${CONFIG_FILE}.d" \
    "${ARG_TARGET[@]}"
