#!/bin/bash




set -e

# Switch to the build directory
cd "$(dirname "${BASH_SOURCE[0]}")"

# Read settings written by bootstrap.bash
source ".bob.bootstrap"

# Switch to the working directory
cd -P "${WORKDIR}"

# Get Bob bootstrap version
source "${BOB_DIR}/bob.bootstrap.version"

if [[ "${BOB_BOOTSTRAP_VERSION}" != "${BOB_VERSION}" ]]; then
    echo "This build directory must be re-bootstrapped. Bob has changed since this output directory was bootstrapped." >&2
    exit 1
fi

# Stop the build if menuconfig.py or update_config.py failed
if [[ -e "${CONFIG_FILE}.error" ]]; then
    echo "Configuration errors are present, the build cannot proceed further." >&2
    exit 1
fi

# Refresh the configuration. This means that options changed or added since the
# last build will be chosen from their defaults automatically, so that users
# don't have to reconfigure manually if the config database changes.
python3 "${BOB_DIR}/config_system/generate_config_json.py" \
       "${CONFIG_FILE}" --database "${SRCDIR}/Mconfig" \
       --json "${CONFIG_JSON}" ${BOB_CONFIG_OPTS}

# Get a hash of the environment so we can detect if we need to
# regenerate the build.ninja
python3 "${BOB_DIR}/scripts/env_hash.py" "${BUILDDIR}/.env.hash"

# If enabled, the following environment variables optimize the performance
# of ccache. Otherwise they have no effect.
# To build with ccache, set the environment variable CCACHE_DIR to where the
# cache is to reside and add CCACHE=y to the build config to enable.
export CCACHE_CPP2=y
export CCACHE_SLOPPINESS=file_macro,time_macros
# Explicitly disable CCACHE_BASEDIR - when it's enabled, ccache will rewrite
# paths in depfiles to be relative to it, but that will cause Ninja to miss
# dependencies on builds where everything else is using absolute paths.
export CCACHE_BASEDIR=

NINJA_ARGS=()
# Newer `ninja` does not have `dupbuild` warning flag
if ! test "${NINJA} -w list | grep -q '^  dupbuild'"; then
  NINJA_ARGS+=( "-w" "dupbuild=err" )
fi

# Build the builder if necessary
BUILDDIR="${BUILDDIR}" SKIP_NINJA=true "${BOB_DIR}/blueprint/blueprint.bash"

# Do the actual build
"${NINJA}" -f "${BUILDDIR}/build.ninja" "${NINJA_ARGS[@]}" "$@"
