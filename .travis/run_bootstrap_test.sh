#!/usr/bin/env bash
set -eE
trap "echo '<------------- run_bootstrap_test.sh failed'" ERR

# Get parent SHA
PARENT=$(git merge-base origin/master ${TRAVIS_COMMIT})

# Check if version update file was changed and if not verify Bob build
if [[ $(git diff --name-only ${PARENT} HEAD | grep "bob.bootstrap.version") ]]; then
    echo "Bob version file has change between parent and current commit. Skipping"
else
    BUILDDIR=${HOME}/bob_build
    git checkout ${PARENT}
    ${BOB_ROOT}/tests/bootstrap
    ${BUILDDIR}/config
    ${BUILDDIR}/buildme bob_tests
    git checkout ${TRAVIS_COMMIT}
    ${BUILDDIR}/buildme bob_tests
fi