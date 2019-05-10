#!/bin/bash
set -eE
trap "echo '<------------- run_build_tests.sh failed'" ERR

export TEST_NON_ASCII_IN_ENV_HASH='ó'
build_dir=build-test
cd ${BOB_ROOT}/tests/
rm -rf ${build_dir} # Cleanup test directory
./bootstrap -o ${build_dir}
# Test by explicitly requesting the `bob_tests` alias, which should include all
# test cases, including alias tests, which can't just set `build_by_default`.
${build_dir}/config && ${build_dir}/buildme bob_tests
