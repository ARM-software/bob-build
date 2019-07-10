#!/usr/bin/env bash
source .travis/utils.sh
STATUS_CODE=0 # reset

####################
fold_start 'Setup:python'
    source .travis/set_python_version.sh
    check_result $? "Set python version:"
fold_end

####################
fold_start 'relative_path_tests.sh'
    bash tests/relative_path_tests.sh
    check_result $? "Relative path tests:"
fold_end
####################

####################
fold_start 'build_tests.sh'
    bash tests/build_tests.sh
    build_result=$?
    check_result ${build_result} "Build tests:"
fold_end
####################

# Tests of go code (python version doesn't matter)
if [ ${DO_GO_TESTS} -eq 1 ] ; then
    ####################
    fold_start 'run_go_tests.sh'
        bash .travis/run_go_tests.sh
        check_result $? "Go tests:"
    fold_end
    ####################
fi

# Tests of python code (go version doesn't matter)
if [ ${DO_PYTHON_TESTS} -eq 1 ] ; then
    ####################
    fold_start 'run_tests.py'
        config_system/tests/run_tests.py
        check_result $? "config_system regression tests:"
    fold_end
    ####################

    ####################
    fold_start 'run_formatter_tests.py'
        config_system/tests/run_tests_formatter.py
        check_result $? "Mconfigfmt tests:"
    fold_end
    ####################

    ####################
    fold_start 'pytest config_system'
        # The newer command `pytest` is not available on Ubuntu 16.04, which the
        # Travis environment uses, so invoke the older `py.test` here.
        py.test-${PYTHON_SUFFIX} config_system
        check_result $? "config_system pytest:"
    fold_end
    ####################
fi

####################
# This test is issued only if build test passed previously.
# We do this last as this test changes the checkout
fold_start 'run_bootstrap_not_required.sh'
    if [[ ${build_result} == 0 ]];then
        bash .travis/run_bootstrap_test.sh
        check_result $? "Bootstrap version test: "
    else
        result_skip "Build tests not passing"
    fi
fold_end

####################

exit $STATUS_CODE
