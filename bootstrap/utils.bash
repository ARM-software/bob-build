# Copyright 2018-2019 Arm Limited.
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

function write_bootstrap() {
    # Always use the host_explore config plugin
    local BOB_CONFIG_PLUGIN_OPTS="-p ${BOB_DIR}/scripts/host_explore"

    # Add any other plugins requested by the caller
    for i in ${BOB_CONFIG_PLUGINS}; do
        BOB_CONFIG_PLUGIN_OPTS="${BOB_CONFIG_PLUGIN_OPTS} -p ${i}"
    done

    BOB_CONFIG_PLUGIN_OPTS="${BOB_CONFIG_PLUGIN_OPTS} -p ${BOB_DIR}/scripts/generate_config_json"

    source "${BOB_DIR}/bob.bootstrap.version"

    sed -e "s|@@WorkDir@@|${WORKDIR}|" \
        -e "s|@@BuildDir@@|${BUILDDIR}|" \
        -e "s|@@SrcDir@@|${SRCDIR}|" \
        -e "s|@@BobDir@@|${BOB_DIR}|" \
        -e "s|@@TopName@@|${TOPNAME}|" \
        -e "s|@@ListFile@@|${BLUEPRINT_LIST_FILE}|" \
        -e "s|@@ConfigName@@|${CONFIGNAME}|" \
        -e "s|@@BobConfigOpts@@|${BOB_CONFIG_OPTS}|" \
        -e "s|@@BobConfigPluginOpts@@|${BOB_CONFIG_PLUGIN_OPTS}|" \
        -e "s|@@BobBootstrapVersion@@|${BOB_VERSION}|" \
        "${BOB_DIR}/bob.bootstrap.in" > "${BUILDDIR}/.bob.bootstrap.tmp"
    rsync -c "${BUILDDIR}/.bob.bootstrap.tmp" "${BUILDDIR}/.bob.bootstrap"
}

function create_config_symlinks() {
    local BOB_DIR="${1}" BUILDDIR="${2}"

    ln -sf "${BOB_DIR}/config.bash" "${BUILDDIR}/config"
    ln -sf "${BOB_DIR}/menuconfig.bash" "${BUILDDIR}/menuconfig"
    ln -sf "${BOB_DIR}/config_system/mconfigfmt.py" "${BUILDDIR}/mconfigfmt"
}

function create_bob_symlinks() {
    local BOB_DIR="${1}" BUILDDIR="${2}"

    ln -sf "${BOB_DIR}/bob.bash" "${BUILDDIR}/bob"
    ln -sf "${BOB_DIR}/bob_graph.bash" "${BUILDDIR}/bob_graph"
}
