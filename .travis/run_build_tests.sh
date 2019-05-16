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

# Helper function for testing that appropriate files are rebuilt after
# a specified source is modified.
function check_dep_updates() {
    local DESC="${1}"
    local DIR="${2}"
    local SRC="${3}"
    shift 3
    local UPDATE=("$@")
    local RESULT=0

    echo "Checking dependency updates for ${DESC}"

    # Pre-flight checks
    if [ ! -e "${SRC}" ] ; then
        echo Error: Source "${SRC}" expected but does not exist
        RESULT=1
    fi
    for file in "${UPDATE[@]}"; do
        if [ ! -e "${file}" ] ; then
            echo Error: Output "${file}" expected but does not exist prior to rebuild
            RESULT=1
        fi
    done
    if [ "${RESULT}" -ne 0 ] ; then
        return ${RESULT}
    fi

    # Wait for a second in case the file system has poor timestamp resolution
    sleep 1
    touch "${SRC}"
    ${DIR}/buildme bob_tests

    for file in "${UPDATE[@]}"; do
        if [ ! -e "${file}" ] ; then
            echo Error: Output "${file}" expected but does not exist after rebuild
            RESULT=1
        elif [ "${file}" -ot "${SRC}" ] ; then
            echo Error: Output "${file}" is older than source "${SRC}" after building
            RESULT=1
        fi
    done

    return ${RESULT}
}

## Various checks that dependency tracking is working. Re-use the
## build-indep build directory from above.
build_dir=build-indep

# library dependencies on source files
SRC=tests/static_libs/a.c
UPDATE=(${build_dir}/target/objects/sl_liba/static_libs/a.c.o
        ${build_dir}/target/static/sl_liba.a
        ${build_dir}/target/executable/sl_main_whole)
check_dep_updates "library sources" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# library dependencies on header file
SRC=tests/static_libs/a.h
UPDATE+=(${build_dir}/target/objects/sl_libb/static_libs/b.c.o
         ${build_dir}/target/static/sl_libb.a)
check_dep_updates "library headers" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# kernel module dependencies on sources
SRC=tests/kernel_module/module/test_module.c
UPDATE=(${build_dir}/target/kernel_modules/test_module/test_module.ko)
check_dep_updates "kernel module source" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# kernel module dependencies on kernel header
SRC=tests/kernel_module/kdir/include/header.h
UPDATE=(${build_dir}/target/kernel_modules/test_module/test_module.ko)
check_dep_updates "kernel headers" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# generated sources
SRC=tests/generate_source/before_generate.in
UPDATE=(${build_dir}/gen/generate_source_single/single.cpp
        ${build_dir}/target/executable/validate_link_generate_sources)
check_dep_updates "generated sources" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# generated sources tool update
SRC=tests/generate_source/gen.sh
UPDATE=(${build_dir}/gen/gen_sources_and_headers/foo/src/foo.c
        ${build_dir}/gen/gen_sources_and_headers/foo/foo.h
        ${build_dir}/gen/gen_sources_and_headers/foo/src/foo.c
        ${build_dir}/target/executable/bin_gen_sources_and_headers)
check_dep_updates "generated source tool" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# generated sources host_bin update
SRC=tests/shared_libs/main.c
UPDATE=(${build_dir}/host/executable/sharedtest
        ${build_dir}/gen/use_sharedtest_host/use_sharedtest_host_main.c
        ${build_dir}/target/executable/use_sharedtest_host_gen_source)
check_dep_updates "generated source host_bin" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# generated sources with depfiles
SRC=tests/generate_source/depgen2.in
UPDATE=(${build_dir}/gen/gen_source_depfiles/output.txt)
check_dep_updates "generate source depfile" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# resource dependencies
SRC=tests/resources/main.c
UPDATE=(${build_dir}/work/bob/linux/y/main.c)
check_dep_updates "resources" "${build_dir}" "${SRC}" "${UPDATE[@]}"
