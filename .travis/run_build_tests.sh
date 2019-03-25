#!/bin/bash
set -eE
trap "echo '<------------- run_build_tests.sh failed'" ERR

export TEST_NON_ASCII_IN_ENV_HASH='ó'
cd ${BOB_ROOT}/tests/
rm -rf build-test # Cleanup test directory
./bootstrap -o build-test
cd build-test
# Test by explicitly requesting the `bob_tests` alias, which should include all
# test cases, including alias tests, which can't just set `build_by_default`.
./config && ./buildme bob_tests
