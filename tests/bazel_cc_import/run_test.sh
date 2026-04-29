#!/usr/bin/env bash
set -eEuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BOB_ROOT="$(cd "${TESTS_DIR}/.." && pwd)"
BUILD_DIR="${1:-build-bazel-import}"
BOB_BUILD_DIR="${BOB_ROOT}/${BUILD_DIR}"
BAZEL_OUTPUT_USER_ROOT="${BOB_BUILD_DIR}.bazel_output_user_root"
GENERATED_BPLIST="${BOB_BUILD_DIR}.bplist"
BAZEL_STARTUP_ARGS=(--output_user_root="${BAZEL_OUTPUT_USER_ROOT}")

cleanup() {
    if [[ -n "${BAZEL:-}" ]]; then
        "${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" shutdown >/dev/null 2>&1 || true
    fi
    rm -rf "${BOB_BUILD_DIR}" "${BAZEL_OUTPUT_USER_ROOT}" "${GENERATED_BPLIST}"
}

trap cleanup EXIT
trap 'echo "<------------- $(basename "${0}") failed"' ERR

require_file() {
    if [[ ! -f "$1" ]]; then
        echo "Expected file does not exist: $1" >&2
        exit 1
    fi
}

require_symlink() {
    if [[ ! -L "$1" ]]; then
        echo "Expected symlink does not exist: $1" >&2
        exit 1
    fi
}

if command -v bazelisk >/dev/null 2>&1; then
    BAZEL=bazelisk
elif command -v bazel >/dev/null 2>&1; then
    BAZEL=bazel
else
    echo "Skipping bazel_import test: neither bazelisk nor bazel is available"
    exit 0
fi

pushd "${BOB_ROOT}" >/dev/null

"${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" clean

"${BAZEL}" "${BAZEL_STARTUP_ARGS[@]}" build \
    --aspects=//tests/bazel_cc_import/bazel:bob_import_cc_aspect.bzl%bob_import_cc_aspect \
    --output_groups=bob_import_cc_bp \
    //tests/bazel_cc_import/bazel/header_only/... \

EXPECTED_BUILD_BPS=(
    bazel-bin/tests/bazel_cc_import/bazel/header_only/includes/tests_bazel_cc_import_bazel_header_only_includes_includes/build.bp
    bazel-bin/tests/bazel_cc_import/bazel/header_only/normal/tests_bazel_cc_import_bazel_header_only_normal_normal/build.bp
    bazel-bin/tests/bazel_cc_import/bazel/header_only/strip_prefix/tests_bazel_cc_import_bazel_header_only_strip_prefix_strip_prefix/build.bp
)

EXPECTED_HEADER_LINKS=(
    bazel-bin/tests/bazel_cc_import/bazel/header_only/includes/tests_bazel_cc_import_bazel_header_only_includes_includes/include/api.h
    bazel-bin/tests/bazel_cc_import/bazel/header_only/normal/tests_bazel_cc_import_bazel_header_only_normal_normal/include/tests/bazel_cc_import/bazel/header_only/normal/api.h
    bazel-bin/tests/bazel_cc_import/bazel/header_only/strip_prefix/tests_bazel_cc_import_bazel_header_only_strip_prefix_strip_prefix/include/nested/api.h
)


for path in "${EXPECTED_BUILD_BPS[@]}"; do
    require_file "${path}"
done

for path in "${EXPECTED_HEADER_LINKS[@]}"; do
    require_symlink "${path}"
done

{
    printf './bazel_cc_import/build.bp\n'
    printf '%s\n' "${EXPECTED_BUILD_BPS[@]}" | sort | sed 's#^#../#'
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

TEST_EXECUTABLES=(
    bob_test_bazel_cc_import_header_only_normal
    bob_test_bazel_cc_import_header_only_includes
    bob_test_bazel_cc_import_header_only_strip_prefix
)

for executable in "${TEST_EXECUTABLES[@]}"; do
    env "${BOB_BUILD_DIR}/target/executable/${executable}"
done

popd >/dev/null
