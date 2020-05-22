#!/usr/bin/env bash

BOB_ROOT=$(dirname ${0})/../..
find "${BOB_ROOT}" -name "*.py" -print0 \
    | xargs -0 pycodestyle --config="${BOB_ROOT}/.pycodestyle"
