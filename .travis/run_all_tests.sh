#!/bin/bash
source .travis/utils.sh
STATUS_CODE=0 # reset

####################
fold_start 'setup_python'
    source ${BOB_ROOT}/.travis/set_python_version.sh
    check_result $? "Check setup_python"
fold_end 'setup_python'

####################
fold_start 'run_build_tests.sh'
    bash ${BOB_ROOT}/.travis/run_build_tests.sh
    build_result = $?
    check_result ${build_result} "Check run_build_tests: "
fold_end
####################

####################
fold_start 'run_go_tests.sh'
    bash ${BOB_ROOT}/.travis/run_go_tests.sh
    check_result $? "Check run_go_tests: "
fold_end
####################

####################
fold_start 'run_tests.sh'
    bash ${BOB_ROOT}/.travis/run_tests.sh
    check_result $? "Check run_tests: "
fold_end
####################

####################
fold_start 'run_formatter_tests.sh'
    bash ${BOB_ROOT}/.travis/run_formatter_tests.sh
    check_result $? "Check run_formatter_tests: "
fold_end
####################

# This test is issued only if build test passed previously
####################

fold_start 'run_bootstrap_not_required.sh'
    if [[ ${build_result} == 0 ]];then
        bash ${BOB_ROOT}/.travis/run_bootstrap_test.sh
        check_result $? "Check run_bootstrap_not_required: "
    else
        result_skip "Build tests not passing"
    fi
fold_end

####################

exit $STATUS_CODE
