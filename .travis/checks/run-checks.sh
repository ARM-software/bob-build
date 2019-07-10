#!/usr/bin/env bash
source .travis/utils.sh
STATUS_CODE=0 # reset

# Useful for debug we should keep this
git log --graph --oneline origin/master...HEAD
echo "----"

if [ ${DO_COMMIT_CHECKS} -eq 1 ]; then
    ####################
    fold_start 'Check:go-vet'
        go vet ./...
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
        python3 .travis/checks/check-signoff.py
        check_result $? "signoff:"
    fold_end
    ####################
fi

exit $STATUS_CODE
