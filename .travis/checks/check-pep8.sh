#!/usr/bin/env bash

BOB_ROOT=$(dirname ${0})/../..
find "${BOB_ROOT}" -name "*.py" -print0 | xargs -0 python -m pep8 --config="${BOB_ROOT}/.pep8"
