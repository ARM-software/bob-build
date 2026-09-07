#!/bin/bash




set -e
trap 'echo "*** Unexpected error ***"' ERR

ORIG_PWD="$(pwd)"

# Move to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")" || exit
source ".bob.bootstrap"

declare -a ARG_TARGET
BAZEL_MODE=false

for arg in "$@"
do
    if [[ ${arg} == --platforms || ${arg} == --platforms=* ]];then
        BAZEL_MODE=true
        break
    fi
done

if ${BAZEL_MODE};then
    if ! command -v bazelisk &> /dev/null;then
        echo "Error: Bazel configuration requires 'bazelisk', but it was not found in PATH." >&2
        echo "Install bazelisk or run config without --platforms to use legacy Bob configuration." >&2
        exit 127
    fi
    if [ -z "${BOB_BAZEL_CONFIG_TARGET}" ];then
        echo "Error: --platforms was provided, but no Bazel configuration target is configured." >&2
        echo "Set BOB_BAZEL_CONFIG_TARGET during bootstrap or run config without --platforms." >&2
        exit 2
    fi

    echo "Using bazel for configuration, all arguments will be forwarded to bazel."
    # In Bazel mode, forward all arguments to bazel without interpreting them.
    BAZEL_ROOT="$(
        cd "${SRCDIR}"
        bazelisk info workspace --ui_event_filters=-info --noshow_progress 2>/dev/null
    )"
    (
        cd "${BAZEL_ROOT}"
        bazelisk build "${BOB_BAZEL_CONFIG_TARGET}" "$@"
    )

    BAZEL_CONFIG="$(
        cd "${BAZEL_ROOT}"
        bazelisk cquery "${BOB_BAZEL_CONFIG_TARGET}" "$@" --output=files
    )"
    if [[ ${BAZEL_CONFIG} != /* ]];then
        BAZEL_CONFIG="${BAZEL_ROOT}/${BAZEL_CONFIG}"
    fi
    if [ ! -f "${BAZEL_CONFIG}" ];then
        echo "Bazel did not create the expected config: ${BAZEL_CONFIG}" >&2
        exit 1
    else
        echo "Bazel config: ${BAZEL_CONFIG}"
    fi
    ARG_TARGET=("${BAZEL_CONFIG}")
else
    for arg in "$@"
    do
        # Preserve configuration assignments (for example, BACKEND_USER=y) for
        # update_config's eval-based command.
        if [[ $arg =~ "=" ]];then
            ARG_TARGET+=(\'"${arg}"\')
        # Absolute paths (for example, /tmp/custom.profile) need no resolution.
        elif [ "${arg:0:1}" == "/" ];then
            ARG_TARGET+=("${arg}")
        else
            # Resolve existing relative files from the caller's working
            # directory; otherwise, treat the argument as a profile name.
            if [ -f "${ORIG_PWD}/${arg}" ];then
                ARG_TARGET+=("${ORIG_PWD}/${arg}")
            else
                ARG_TARGET+=("${SRCDIR}/bldsys/profiles/${arg}")
            fi
        fi
    done
fi

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
