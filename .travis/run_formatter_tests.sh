#!/usr/bin/env bash
set -eE
trap "echo '<------------- run_formatter_tests.sh failed'" ERR

cd ${BOB_ROOT}/config_system/tests
which python

# Execute run_test_formatter
python run_tests_formatter.py
