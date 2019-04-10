#!/usr/bin/env bash
set -eE
trap "echo '<------------- run_bootstrap_test.sh failed'" ERR

# Make sure that commands are executed in the right directory
cd ${TRAVIS_BUILD_DIR}

PARENT=$(git merge-base origin/master ${TRAVIS_COMMIT})

# Check if version update file was changed and if not verify Bob build
if [[ $(git diff --name-only ${PARENT} ${TRAVIS_COMMIT} | grep "bob.bootstrap.version") ]]; then
    echo "Bob version file has change between parent and current commit. Skipping verification step"
else
    build_dir=${HOME}/bob_build
    git checkout ${PARENT}
    cd ${BOB_ROOT}/tests/ && ./bootstrap -o ${build_dir}
    ${build_dir}/config
    ${build_dir}/buildme bob_tests
    cd ${TRAVIS_BUILD_DIR}
    git checkout ${TRAVIS_COMMIT}
    rm ${build_dir}/build.ninja
    ${build_dir}/buildme bob_tests
fi
