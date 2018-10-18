#!/bin/bash
set -eE
trap "echo '<------------- run_build_tests.sh failed'" ERR

cd ${BOB_ROOT}/tests/
rm -rf build-test # Cleanup test directory
BUILDDIR=build-test ./bootstrap
cd build-test
./config && ./buildme
