#!/bin/bash




# SRCDIR - Path to base of source tree. This can be relative to PWD or absolute.
# BUILDDIR - Build output directory. This can be relative to PWD or absolute.
# CONFIGDIR - Configuration directory. This can be relative to PWD or absolute.
# CONFIGNAME - Name of the configuration file.
# BLUEPRINT_LIST_FILE - Path to file listing all Blueprint input files.
#                       This can be relative to PWD or absolute.
# BOB_CONFIG_OPTS - Configuration options to be used when calling the
#                   configuration system.
# BOB_CONFIG_PLUGINS - Configuration system plugins to use

# The location that this script is called from determines the working
# directory of the build.

set -e

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")

source "${SCRIPT_DIR}/pathtools.bash"
source "${SCRIPT_DIR}/bootstrap/utils.bash"

if ! command -v go &> /dev/null
then
    >&2 echo "ERROR: Go is required for Bob, please install and try again."
    exit 1
fi

if [ -z "${GO120_NO_STD_INSTALL-}" ]; then
  GO120_NO_STD_INSTALL=0
fi

go_version=$(go version | { read _ _ v _; echo "${v#go}"; })
if [ "$(printf "%s\n1.20\n" "$go_version" | sort -t '.' -k 1,1 -k 2,2 -g | head -n 1)" == "1.20" ]; then
  # Since go 1.20, std modules are not installed by default. This breaks Blueprint.
  # Pre-install these modules before proceeding.
  # Should the GOPATH change or be deleted this script may have to be re-run.
  if [ ${GO120_NO_STD_INSTALL} = 0 ]; then
    echo 'WARNING: Go >=1.20 is not fully supported by Bob/Blueprint and requires a pre-installed Go `std` package on the system. Attempting workaround. You may disable this workaround by setting `GO120_NO_STD_INSTALL=1`'

    ret=0
    (GODEBUG=installgoroot=all go install std) || ret=$?

    if [ $ret -eq 126 ]; then
      echo 'WARNING: Permissions failed when installing Go standard library. Build will proceed assuming a sandboxed envrioment, if you are not in a sandbox with a pre-existing Go `std` library installed please run `GODEBUG=installgoroot=all go install std` with sudo.'
    elif [ $ret -ne 0 ]; then
      echo 'ERROR: Unexpected error installing Go `std` package for Go >=1.20. Error code:' $ret
      exit $ret
    fi
  fi
fi

# Use defaults where we can. Generally the caller should set these.
if [ -z "${SRCDIR}" ] ; then
    # If not specified, assume the current directory
    export SRCDIR=.
fi

if [[ -z "$BUILDDIR" ]]; then
  echo "BUILDDIR is not set - using ."
  export BUILDDIR=.
fi

if [[ -z "$CONFIGDIR" ]]; then
  CONFIGDIR="${BUILDDIR}"
else
  mkdir -p "${CONFIGDIR}"
fi

if [[ -z "$CONFIGNAME" ]]; then
  echo "CONFIGNAME is not set - using bob.config"
  CONFIGNAME="bob.config"
fi

if [[ -z "$BOB_CONFIG_OPTS" ]]; then
  BOB_CONFIG_OPTS=""
fi

if [[ -z "$BOB_CONFIG_PLUGINS" ]]; then
  BOB_CONFIG_PLUGINS=""
fi

if [ "${BUILDDIR}" = "." ] ; then
    WORKDIR=.
else
    # Create the build directory
    mkdir -p "$BUILDDIR"

    # Relative path from build directory to working directory
    WORKDIR=$(relative_path "${BUILDDIR}" "$(pwd)")
    export WORKDIR
fi

BOOTSTRAP_GLOBFILE="${BUILDDIR}/.bootstrap/build-globs.ninja"
if [ -f "${BOOTSTRAP_GLOBFILE}" ]; then
    PREV_DIR=$(sed -n -e "s/^g.bootstrap.buildDir = \(.*\)/\1/p" "${BOOTSTRAP_GLOBFILE}")
    if [ "${PREV_DIR}" != "${BUILDDIR}" ] ; then
        # BOOTSTRAP_GLOBFILE is invalid if BUILDDIR has changed
        # Invalidate it so that the bootstrap builder can be built
        cat /dev/null > "${BOOTSTRAP_GLOBFILE}"
        # On OSX, also force a rebuild of microfactory
        if [ "$(uname)" = "Darwin" ] ; then
            rm -f "${BUILDDIR}/.minibootstrap/microfactory_$(uname)"
        fi
    fi
fi

# Calculate Bob directory relative to the working directory.
BOB_DIR="$(relative_path "$(pwd)" "${SCRIPT_DIR}")"
CONFIG_FILE="${CONFIGDIR}/${CONFIGNAME}"
CONFIG_JSON="${CONFIGDIR}/.bob.config.json"

# Bob warnings log file
BOB_LOG_WARNINGS_FILE="${BUILDDIR}/.bob.warnings.csv"

# space separated values, e.g. "*:W RelativeUpLinkWarning:E"
BOB_LOG_WARNINGS="DeprecatedFilegroupSrcs:W"

export BOB_DIR
export CONFIG_FILE
export CONFIG_JSON
export BOB_LOG_WARNINGS
export BOB_LOG_WARNINGS_FILE
export TOPNAME="build.bp"
export BOOTSTRAP="${BOB_DIR}/bootstrap.bash"
export BLUEPRINTDIR="${BOB_DIR}/blueprint"

# Bootstrap blueprint.
"${BLUEPRINTDIR}/bootstrap.bash"

# Configure Bob in the build directory
write_bootstrap

if [ ${SRCDIR:0:1} != '/' ]; then
    # Use relative symlinks
    BOB_DIR_FROM_BUILD="$(relative_path "$(bob_realpath "${BUILDDIR}")" "${SCRIPT_DIR}")"
else
    # Use absolute symlinks
    BOB_DIR_FROM_BUILD="$(bob_realpath "${SCRIPT_DIR}")"
fi
create_config_symlinks "${BOB_DIR_FROM_BUILD}" "${BUILDDIR}"
create_bob_symlinks "${BOB_DIR_FROM_BUILD}" "${BUILDDIR}"
