#!/usr/bin/env bash
set -eE
trap "echo '<------------- run_tests.sh failed'" ERR

cd ${BOB_ROOT}/config_system/tests
which python

# Execute run_test
python run_tests.py
