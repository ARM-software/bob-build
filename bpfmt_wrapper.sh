#!/bin/bash
set -e
WORK_DIR=$(pwd)
if test "${BUILD_WORKING_DIRECTORY+x}"; then
  cd "$BUILD_WORKING_DIRECTORY"
fi
"$WORK_DIR"/bpfmt_/bpfmt "${@:1}"
