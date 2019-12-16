#!/usr/bin/env bash
set -eE
trap "echo '<------------- $(basename ${0}) failed'" ERR

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")
BOB_ROOT="${SCRIPT_DIR}/.."

# File must be present
function check_installed() {
    local FILE="${1}"

    [ -f "${FILE}" ] || { echo "${FILE} not installed" ; false; }
}

# File must be stripped
function check_stripped() {
    local FILE="${1}"
    local OS="${2}"

    if [ "$OS" != "OSX" ] ; then
        [ $(nm -a "${FILE}" | wc -l) = "0" ] || { echo "${FILE} not stripped" ; false; }
    else
        # The symbol below is always expected on macOS
        [ $(nm -a "${FILE}" | grep -Ev " dyld_stub_binder$" | wc -l) = "0" ] || {
            echo "${FILE} not stripped"
            false
        }
    fi
}

case "$(uname -s)" in
    Darwin*)
        OS=OSX
        SHARED_LIBRARY_EXTENSION=".dylib"
        ;;
    *)
        OS=LINUX
        SHARED_LIBRARY_EXTENSION=".so"
        ;;
esac

OPTIONS="$OS=y"

# Do simple checks on the output of each build
function check_build_output() {
    local DIR="${1}"
    shift

    echo "Checking build output under ${DIR}"

    # Check that installed libraries/binaries are present
    check_installed "${DIR}/install/lib/libstripped_library${SHARED_LIBRARY_EXTENSION}"
    check_installed "${DIR}/install/bin/stripped_binary"

    # The stripped library must not contain symbols
    check_stripped "${DIR}/install/lib/libstripped_library${SHARED_LIBRARY_EXTENSION}" "$OS"
    check_stripped "${DIR}/install/bin/stripped_binary" "$OS"
}

export TEST_NON_ASCII_IN_ENV_HASH='ó'

pushd "${BOB_ROOT}" &> /dev/null

TEST_DIRS=("build-indep"
           "build-in-outp"
           "tests/build-in-src")
rm -rf "${TEST_DIRS[@]}"

# Test by explicitly requesting the `bob_tests` alias, which should include all
# test cases, including alias tests, which can't just set `build_by_default`.

# Build with working directory in source directory
build_dir=build-in-src
pushd "tests" &> /dev/null
./bootstrap -o ${build_dir}
${build_dir}/config ${OPTIONS} && ${build_dir}/buildme bob_tests
check_build_output "${build_dir}"
popd &> /dev/null

# Build in an independent working directory
build_dir=build-indep
tests/bootstrap -o ${build_dir}
${build_dir}/config ${OPTIONS} && ${build_dir}/buildme bob_tests
check_build_output "${build_dir}"

# Build with the working directory in the output directory
build_dir=build-in-outp
mkdir ${build_dir}
pushd ${build_dir} &> /dev/null
../tests/bootstrap -o .
./config ${OPTIONS} && ./buildme bob_tests
popd &> /dev/null
check_build_output "${build_dir}"

# A re-bootstrapped build directory with a different working directory
# should still work. Re-use the last directory
echo Checking rebootstrap
tests/bootstrap -o ${build_dir}
${build_dir}/buildme bob_tests

# Check static archives are built from scratch. Re-use the last directory
echo Reconfiguring to check archives are clean
${build_dir}/config ${OPTIONS} STATIC_LIB_TOGGLE=y
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
UPDATE=(${build_dir}/gen/gen_source_depfile/output.txt)
check_dep_updates "generate source depfile" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# resource dependencies
SRC=tests/resources/main.c
UPDATE=(${build_dir}/work/bob/linux/y/main.c)
check_dep_updates "resources" "${build_dir}" "${SRC}" "${UPDATE[@]}"

# implicit output
SRC=tests/implicit_outs/input.in
UPDATE=(${build_dir}/target/executable/build_implicit_out
        ${build_dir}/target/executable/include_implicit_header)
check_dep_updates "implicit output" "${build_dir}" "${SRC}" "${UPDATE[@]}"

if [ "$OS" != "OSX" ] ; then
    # kernel module dependencies on sources
    SRC=tests/kernel_module/module1/test_module1.c
    UPDATE=(${build_dir}/target/kernel_modules/test_module1/test_module1.ko)
    check_dep_updates "kernel module source" "${build_dir}" "${SRC}" "${UPDATE[@]}"

    # kernel module dependencies on kernel header
    SRC=tests/kernel_module/kdir/include/kernel_header.h
    UPDATE=(${build_dir}/target/kernel_modules/test_module1/test_module1.ko)
    check_dep_updates "kernel headers" "${build_dir}" "${SRC}" "${UPDATE[@]}"
fi

# Clean up
rm -rf "${TEST_DIRS[@]}"
popd &> /dev/null
