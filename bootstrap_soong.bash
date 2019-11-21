#!/usr/bin/env bash

# Copyright 2019 Arm Limited.
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

set -eu
trap 'echo "*** Unexpected error in $0 ***"' ERR

BOB_DIR=$(dirname "${BASH_SOURCE[0]}")

function die {
    echo "${BASH_SOURCE[0]}: ${*}"
    exit 1
}

# ${VAR:-} will substitute an empty string if the variable is unset, which
# stops `set -u` complaining before `die` is invoked.
[[ -z ${SRCDIR:-} ]] && die "\$SRCDIR not set"
[[ -z ${SRCDIR:-} ]] && die "\$SRCDIR not set"
[[ -z ${PROJ_NAME:-} ]] && die "\$PROJ_NAME not set"
[[ -z ${OUT:-} ]] && die "\$OUT not set - did you run envsetup.sh and lunch?"

[[ -e ${SRCDIR}/Android.mk ]] && die "${SRCDIR}/Android.mk conflicts with Android.bp. Please remove!"
[[ -f "build/make/core/envsetup.mk" ]] || die "Working dir must be the root of an Android build tree"

# The following variables are optional - give them empty default values.
BOB_CONFIG_OPTS="${BOB_CONFIG_OPTS-}"
BOB_CONFIG_PLUGINS="${BOB_CONFIG_PLUGINS-}"

source "${BOB_DIR}/pathtools.bash"
source "${BOB_DIR}/bootstrap/utils.bash"

# TODO: Generate the config file based on the command-line arguments
BUILDDIR="${OUT}/gen/STATIC_LIBRARIES/${PROJ_NAME}-config"
mkdir -p "${BUILDDIR}"

CONFIG_FILE="${BUILDDIR}/${CONFIGNAME}"
CONFIG_JSON="${BUILDDIR}/config.json"

WORKDIR="$(pwd)" write_bootstrap

# Create symlinks to the config system wrapper scripts
create_config_symlinks "$(relative_path "${BUILDDIR}" "${BOB_DIR}")" "${BUILDDIR}"

# Create a Go file containing the path to the config file, which will be
# compiled into the Soong plugin. This is required because the module factories
# do not have access to the Soong context when they are called, even though the
# config file must be loaded before then.
SOONG_CONFIG_GO="${BUILDDIR}/soong_config.go"
TMP_GO_CONFIG=$(mktemp)
sed -e "s#@@BOB_CONFIG_OPTS@@#${BOB_CONFIG_OPTS}#" \
    -e "s#@@BOB_DIR@@#${BOB_DIR}#" \
    -e "s#@@BUILDDIR@@#${BUILDDIR}#" \
    -e "s#@@CONFIG_FILE@@#${CONFIG_FILE}#" \
    -e "s#@@CONFIG_JSON@@#${CONFIG_JSON}#" \
    -e "s#@@SRCDIR@@#${SRCDIR}#" \
    "${BOB_DIR}/core/soong_config.go.in" > "${TMP_GO_CONFIG}"
rsync --checksum "${TMP_GO_CONFIG}" "${SOONG_CONFIG_GO}"
rm -f "${TMP_GO_CONFIG}"

SOONG_CONFIG_GO_FROM_BOB=$(relative_path "${BOB_DIR}" "${SOONG_CONFIG_GO}")

# Set up Bob's Android.bp
pushd "${BOB_DIR}" >&/dev/null
TMP_ANDROID_BP=$(mktemp)
sed -e "s#@@PROJ_NAME@@#${PROJ_NAME}#" \
    -e "s#@@SOONG_CONFIG_GO@@#${SOONG_CONFIG_GO_FROM_BOB}#" \
    Android.bp.in > "${TMP_ANDROID_BP}"
rsync --checksum "${TMP_ANDROID_BP}" Android.bp
rm -f "${TMP_ANDROID_BP}"
popd >&/dev/null

# Create an `Android.bp` symlink for every `build.bp` present in the source
# dir, and remove dead links.
pushd "${SRCDIR}" >&/dev/null
find -name build.bp -execdir ln -fs build.bp Android.bp \;
find -name Android.bp -xtype l -delete
popd >&/dev/null
