


function write_bootstrap() {
    # Always use the host_explore config plugin
    local BOB_CONFIG_PLUGIN_OPTS="-p ${BOB_DIR}/scripts/host_explore"

    # Add any other plugins requested by the caller
    # Split ':' separated paths and store them in a PLUGINS array
    IFS=':' read -ra PLUGINS <<< "$BOB_CONFIG_PLUGINS"
    for i in "${PLUGINS[@]}"; do
        BOB_CONFIG_PLUGIN_OPTS="${BOB_CONFIG_PLUGIN_OPTS} -p \'${i}\'"
    done

    source "${BOB_DIR}/bob.bootstrap.version"

    sed -e "s|@@WorkDir@@|${WORKDIR}|" \
        -e "s|@@BuildDir@@|${BUILDDIR}|" \
        -e "s|@@SrcDir@@|${SRCDIR}|" \
        -e "s|@@BobDir@@|${BOB_DIR}|" \
        -e "s|@@TopName@@|${TOPNAME}|" \
        -e "s|@@ListFile@@|${BLUEPRINT_LIST_FILE}|" \
        -e "s|@@ConfigFile@@|${CONFIG_FILE}|" \
        -e "s|@@ConfigJson@@|${CONFIG_JSON}|" \
        -e "s|@@BobConfigOpts@@|${BOB_CONFIG_OPTS}|" \
        -e "s|@@BobConfigPluginOpts@@|${BOB_CONFIG_PLUGIN_OPTS}|" \
        -e "s|@@BobBootstrapVersion@@|${BOB_VERSION}|" \
        -e "s|@@BobLogWarningsFile@@|${BOB_LOG_WARNINGS_FILE}|" \
        -e "s|@@BobMetaFile@@|${BOB_META_FILE}|" \
        -e "s|@@BobLogWarnings@@|${BOB_LOG_WARNINGS}|" \
        "${BOB_DIR}/bob.bootstrap.in" > "${BUILDDIR}/.bob.bootstrap.tmp"
    rsync -c "${BUILDDIR}/.bob.bootstrap.tmp" "${BUILDDIR}/.bob.bootstrap"
}

function create_config_symlinks() {
    local BOB_DIR="${1}" BUILDDIR="${2}"

    ln -sf "${BOB_DIR}/config.bash" "${BUILDDIR}/config"
    ln -sf "${BOB_DIR}/reconfig.bash" "${BUILDDIR}/reconfig"
    ln -sf "${BOB_DIR}/menuconfig.bash" "${BUILDDIR}/menuconfig"
    ln -sf "${BOB_DIR}/print_user_config.bash" "${BUILDDIR}/print_user_config"
    ln -sf "${BOB_DIR}/config_system/mconfigfmt.py" "${BUILDDIR}/mconfigfmt"
}

function create_bob_symlinks() {
    local BOB_DIR="${1}" BUILDDIR="${2}"

    ln -sf "${BOB_DIR}/bob.bash" "${BUILDDIR}/bob"
    ln -sf "${BOB_DIR}/bob_graph.bash" "${BUILDDIR}/bob_graph"
}

function apply_blueprint_patches() {
    local BOB_DIR="${1}"

    # blueprint patches
    declare -a patches=(
        "../patches/blueprint/0001-feat-visit-modules-with-position.patch"
        "../patches/blueprint/0003-fix-remove-dupbuild-flag.patch"
    )

    pushd "${BOB_DIR}" > /dev/null || return

    # reset `blueprint` submodule
    git submodule update --init --recursive

    pushd "blueprint" > /dev/null || return

    # apply patches
    for p in "${patches[@]}"
    do
        git am -q < "${p}"
    done

    popd > /dev/null || return
    popd > /dev/null || return
}
