#!/bin/bash
set -eE
trap "echo '<------------- run_build_tests.sh failed'" ERR

cd ${BOB_ROOT}/tests/
rm -rf build-test # Cleanup test directory
BUILDDIR=build-test ./bootstrap
cd build-test
# Test by explicitly requesting the `bob_tests` alias, which should include all
# test cases, including alias tests, which can't just set `build_by_default`.
./config && ./buildme bob_tests
