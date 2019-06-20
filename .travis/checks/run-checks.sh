#!/bin/bash
source .travis/utils.sh
STATUS_CODE=0 # reset

####################
fold_start 'Check:go-vet'
    bash .travis/checks/check-go-vet.sh
    check_result $? "go vet:"
fold_end
####################

####################
fold_start 'Check:gofmt'
    bash .travis/checks/check-code-format.sh
    check_result $? "gofmt:"
fold_end
####################

####################
fold_start 'Check:pep8'
    bash .travis/checks/check-pep8.sh
    check_result $? "pep8:"
fold_end
####################

####################
fold_start 'Check:pylint'
    bash .travis/checks/check-pylint.sh
    check_result $? "pylint:"
fold_end
####################

####################
fold_start 'Check:copyright'
    bash .travis/checks/check-copyright.sh
    check_result $? "copyright:"
fold_end
####################

####################
fold_start 'Check:signoff'
    git log  --pretty=oneline | head -n 10 # Very useful for debug we should keep this
    echo "----"
    python3 .travis/checks/check-signoff.py
    check_result $? "signoff:"
fold_end
####################

exit $STATUS_CODE
