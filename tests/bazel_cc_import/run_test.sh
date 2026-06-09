#!/usr/bin/env bash
set -eEuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BOB_ROOT="$(cd "${TESTS_DIR}/.." && pwd)"
BUILD_DIR="${1:-build-bazel-import}"
BOB_BUILD_DIR="${BOB_ROOT}/${BUILD_DIR}"
GENERATED_BPLIST="${BOB_BUILD_DIR}.bplist"
BAZEL_STARTUP_ARGS=()

cleanup() {
    if [[ -n "${BAZEL:-}" ]]; then
        "${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" shutdown >/dev/null 2>&1 || true
    fi
    chmod -R u+w "${BOB_BUILD_DIR}" "${GENERATED_BPLIST}" >/dev/null 2>&1 || true
    rm -rf "${BOB_BUILD_DIR}" "${GENERATED_BPLIST}"
}

trap cleanup EXIT
trap 'echo "<------------- $(basename "${0}") failed"' ERR


if command -v bazelisk >/dev/null 2>&1; then
    BAZEL=bazelisk
elif command -v bazel >/dev/null 2>&1; then
    BAZEL=bazel
else
    echo "Skipping bazel_cc_import test: neither bazelisk nor bazel is available"
    exit 0
fi

pushd "${BOB_ROOT}" >/dev/null

"${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" clean

BAZEL_TARGETS=(
    //tests/bazel_cc_import/bazel:test
)

"${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" build "${BAZEL_TARGETS[@]}"

assert_bazel_build_fails() {
    local target="$1"
    local expected="$2"
    local log_file
    log_file="$(mktemp)"

    echo "Checking unsupported Bazel import: ${target}"

    if "${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" build "${target}" >"${log_file}" 2>&1; then
        echo "Expected '${target}' to fail" >&2
        rm -f "${log_file}"
        exit 1
    fi

    if ! grep -Fq "${expected}" "${log_file}"; then
        echo "Expected '${target}' failure to contain '${expected}', got:" >&2
        cat "${log_file}" >&2
        rm -f "${log_file}"
        exit 1
    fi

    rm -f "${log_file}"
}

assert_bazel_build_fails \
    //tests/bazel_cc_import/bazel:dynamic_deps_unsupported_test \
    "dynamic dependencies are not supported for Bob imports"

assert_bazel_build_fails \
    //tests/bazel_cc_import/bazel:multi_output_unsupported_test \
    "Bob import generation supports one output per Bazel target"

mapfile -t GENERATED_BUILD_BPS < <(
    find bazel-bin/tests/bazel_cc_import/bazel -type f -name build.bp | sort
)

if [[ ${#GENERATED_BUILD_BPS[@]} -eq 0 ]]; then
    echo "No generated build.bp files found under bazel-bin/tests/bazel_cc_import/bazel" >&2
    exit 1
fi


{
    printf './bazel_cc_import/build.bp\n'
    printf '%s\n' "${GENERATED_BUILD_BPS[@]}" | sed 's#^#../#'
    printf './bob/Blueprints\n'
    printf './bob/blueprint/Blueprints\n'
} > "${GENERATED_BPLIST}"

source "${TESTS_DIR}/bootstrap_utils.sh"
create_link .. "${TESTS_DIR}/bob"

rm -rf "${BOB_BUILD_DIR}"
export CONFIGNAME="bob.config"
export SRCDIR="${TESTS_DIR}"
export BUILDDIR="${BOB_BUILD_DIR}"
export BLUEPRINT_LIST_FILE="${GENERATED_BPLIST}"
export BOB_LOG_WARNINGS_FILE="${BOB_BUILD_DIR}/.bob.warnings.csv"
export BOB_META_FILE="${BOB_BUILD_DIR}/.bob.meta.json"
export BOB_LOG_WARNINGS=""
export BOB_CONFIG_PLUGINS="${TESTS_DIR}/plugins/test_plugin"

"${BOB_ROOT}/bootstrap_linux.bash"
ln -sf "bob" "${BOB_BUILD_DIR}/buildme"
"${BOB_BUILD_DIR}/config"
"${BOB_BUILD_DIR}/buildme" bob_test_bazel_import

IMPORTED_TOOL_OUTPUT="${BOB_BUILD_DIR}/gen/bob_test_bazel_cc_import_binary_genrule_tool/bazel_imported_tool_output.txt"
if [[ "$(<"${IMPORTED_TOOL_OUTPUT}")" != "42" ]]; then
    echo "Expected imported Bazel tool exit code to be 42, got: $(<"${IMPORTED_TOOL_OUTPUT}")" >&2
    exit 1
fi

TEST_EXECUTABLES=(
    bob_test_bazel_cc_import_header_only_normal
    bob_test_bazel_cc_import_header_only_includes
    bob_test_bazel_cc_import_header_only_strip_prefix
    bob_test_bazel_cc_import_library_simple_shared
    bob_test_bazel_cc_import_library_simple_static
    bob_test_bazel_cc_import_library_dependency_shared
    bob_test_bazel_cc_import_library_dependency_static
)

for executable in "${TEST_EXECUTABLES[@]}"; do
    env "${BOB_BUILD_DIR}/target/executable/${executable}"
done

popd >/dev/null
