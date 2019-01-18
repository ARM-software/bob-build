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
    check_result $? "Check run_build_tests: "
fold_end
####################

####################
fold_start 'run_go_tests.sh'
    bash ${BOB_ROOT}/.travis/run_go_tests.sh
    check_result $? "Check run_go_tests: "
fold_end
####################

exit $STATUS_CODE
