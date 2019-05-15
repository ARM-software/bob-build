#!/bin/bash
set -eE
trap "echo '<------------- run_build_tests.sh failed'" ERR

export TEST_NON_ASCII_IN_ENV_HASH='ó'

# Test by explicitly requesting the `bob_tests` alias, which should include all
# test cases, including alias tests, which can't just set `build_by_default`.

# Build with working directory in source directory
build_dir=build-in-src
cd "${BOB_ROOT}/tests"
rm -rf ${build_dir} # Cleanup test directory
./bootstrap -o ${build_dir}
${build_dir}/config && ${build_dir}/buildme bob_tests

# Build in an independent working directory
build_dir=build-indep
cd "${BOB_ROOT}"
rm -rf ${build_dir} # Cleanup test directory
tests/bootstrap -o ${build_dir}
${build_dir}/config && ${build_dir}/buildme bob_tests

# Build with the working directory in the output directory
build_dir=build-in-outp
cd "${BOB_ROOT}"
rm -rf ${build_dir} # Cleanup test directory
mkdir ${build_dir}
cd ${build_dir}
../tests/bootstrap -o .
./config && ./buildme bob_tests

# A re-bootstrapped build directory with a different working directory
# should still work. Re-use the last directory
cd "${BOB_ROOT}"
tests/bootstrap -o ${build_dir}
${build_dir}/buildme bob_tests
