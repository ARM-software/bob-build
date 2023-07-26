#!/bin/bash




# Check that relative_path() works as expected
SCRIPT_DIR=$(dirname "$0")
BOB_DIR="${SCRIPT_DIR}/.."

source "${BOB_DIR}/pathtools.bash"

HAVE_FAILURE=0

function test_relpath() {
    local SOURCE="${1}"
    local TARGET="${2}"
    local EXPECTED="${3}"
    local STDERR="${4}"
    local RESULT=

    if [ -n "${STDERR}" ]; then
        RESULT="$(relative_path "${SOURCE}" "${TARGET}" 2>&1)"
        EXPECTED="${STDERR}"
    else
        RESULT="$(relative_path "${SOURCE}" "${TARGET}")"
    fi

    if [ "${RESULT}" != "${EXPECTED}" ] ; then
        echo FAIL: relative_path "${SOURCE}" "${TARGET}" expected to return "${EXPECTED}", got "${RESULT}"
        HAVE_FAILURE=1
    fi
}

TEST_DIR="$(mktemp -d -t relative_path_tests.XXXXXX)"
pushd "${TEST_DIR}" >&/dev/null || exit

mkdir -p "a/b/c/d"
mkdir -p "a/b2/c"
mkdir -p "a/b/g"
mkdir -p "a/e/g"
mkdir -p "x/y/z"

# Directory is the same
test_relpath "a"     "a"     "."
test_relpath "a/b"   "a/b"   "."
test_relpath "a/b/c" "a/b/c" "."

# Target is a subdirectory
test_relpath "a"   "a/b"   "b"
test_relpath "a"   "a/b/c" "b/c"
test_relpath "a/b" "a/b/c" "c"
test_relpath "a"   "a/e"   "e"
test_relpath "a"   "a/e/g" "e/g"
test_relpath "a/e" "a/e/g" "g"

# Target is a parent
test_relpath "a/b/c" "a"   "../.."
test_relpath "a/b/c" "a/b" ".."
test_relpath "a/e/g" "a"   "../.."
test_relpath "a/e/g" "a/e" ".."

# Target shares a common parent
test_relpath "a/b"   "a/e"   "../e"
test_relpath "a/b/c" "a/e/g" "../../e/g"
test_relpath "a/b/c" "a/b/g" "../g"

# No shared path (actually this shares the current dir)
test_relpath "a/b/c" "x/y/z" "../../../x/y/z"

# Check directory substring mismatches
test_relpath "a/b" "a/b2/c"  "../b2/c"
test_relpath "a/b2/c" "a/b"  "../../b"

# Check the special case where the common root is `/`
test_relpath "/usr" "/bin" "../bin"
test_relpath "/usr/local/bin" "/bin/bash" "../../../bin/bash"
test_relpath "/" "/bin/bash" "bin/bash"
# On merged-usr systems, `/bin` is a symlink to `/usr/bin`, so needs an extra
# `..` for this case.
[[ -L /bin ]] && test_relpath "/bin" "/" "../.." || test_relpath "/bin" "/" ".."

# Test when the source path contains symlinks. This is a non-trivial part of
# the implementation, because using `..` on symlinked directory will return the
# parent dir of the _target_, not the link. When this is the case, some
# symlinks may need to be expanded.
ln -s "a/b" "sym"

test_relpath "sym" "a/b/c" "c"
test_relpath "sym/c" "sym/c/d" "d"
test_relpath "sym/c/d" "sym/c" ".."
test_relpath "sym" "." "../.."

# Test with source paths containing multiple symlinks. The top-level
# `multiple_links` directory should generally _not_ be expanded in these test
# cases, but the symlinks inside it are.
mkdir -p "multiple_links_dir/first_linked_dir"
mkdir -p "multiple_links_dir/first_linked_dir/a"
mkdir -p "multiple_links_dir/first_linked_dir/x/y/z"
ln -s "multiple_links_dir/first_linked_dir" "multiple_links"
ln -s "x/y" "multiple_links/sym_y"

test_relpath "multiple_links" "multiple_links/sym_y/z" "sym_y/z"
test_relpath "multiple_links/sym_y" "multiple_links" "../.."
test_relpath "multiple_links/sym_y/z" "multiple_links" "../../.."
test_relpath "multiple_links/sym_y/z" "multiple_links/a" "../../../a"
test_relpath "multiple_links/sym_y/z" "sym/c" "../../../../../sym/c"
test_relpath "multiple_links" "multiple_links" "."
test_relpath "multiple_links" "multiple_links_dir/first_linked_dir/x/y" "x/y"
test_relpath "multiple_links_dir" "multiple_links/x" "../multiple_links/x"

# Check that nothing goes horribly wrong with recursive symlinks
ln -s "cycle1" "cycle2"
ln -s "cycle2" "cycle1"
test_relpath "cycle1" "cycle2" "" "relative_path: Source path 'cycle1' does not exist"

# Create a directory of symlinks all pointing to their parent dir, which means
# we can construct paths within it containing any combination of the directory
# names.
mkdir "combinations"
for i in a b c; do
    mkdir "combinations/${i}"
    for j in a b c; do
        ln -s ".." "combinations/${i}/${j}"
    done
done

test_relpath "combinations/c/b/a" "combinations/b" "../b"
test_relpath "combinations/c/a" "combinations/b/c/a/b" "b/c/a/b"

# Cleanup
popd >&/dev/null || exit
rm -rf "${TEST_DIR}"

exit ${HAVE_FAILURE}
