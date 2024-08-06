#!/bin/bash
set -e
WORK_DIR=$(pwd)
if test "${BUILD_WORKING_DIRECTORY+x}"; then
  cd "$BUILD_WORKING_DIRECTORY"
fi
"$WORK_DIR"/external/com_github_google_blueprint/bpfmt/bpfmt_/bpfmt "${@:1}"
