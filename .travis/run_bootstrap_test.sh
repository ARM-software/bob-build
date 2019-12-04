#!/usr/bin/env bash
set -eE
trap "echo '<------------- $(basename ${0}) failed'" ERR

SCRIPT_DIR=$(dirname $0)
BOB_ROOT="${SCRIPT_DIR}/.."

COMMIT=$(git rev-parse HEAD)
PARENT=$(git merge-base origin/master ${COMMIT})

case "$(uname -s)" in
    Darwin*)
        OS=OSX
        ;;
    *)
        OS=LINUX
        ;;
esac

OPTIONS="$OS=y"

# Check if version update file was changed and if not verify Bob build
if git diff --name-only ${PARENT} ${COMMIT} | grep -q "bob.bootstrap.version" ; then
    echo "Bob version file has changed between parent and current commit. Skipping verification step"
else
    build_dir=bootstrap_test
    git checkout ${PARENT}
    cd "${BOB_ROOT}/tests"
    rm -rf ${build_dir} # Cleanup test directory
    ./bootstrap -o ${build_dir}
    ${build_dir}/config ${OPTIONS} && ${build_dir}/buildme bob_tests

    # Wait for filesystems with low timestamp resolution
    sleep 1

    git checkout ${COMMIT}
    rm ${build_dir}/build.ninja
    ${build_dir}/buildme bob_tests
fi
