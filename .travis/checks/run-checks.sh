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
    fold_end $?
    ####################

    ####################
    fold_start 'Check:gofmt'
        bash .travis/checks/check-code-format.sh
    fold_end $?
    ####################

    ####################
    fold_start 'Check:pycodestyle'
        bash .travis/checks/check-pycodestyle.sh
    fold_end $?
    ####################

    ####################
    fold_start 'Check:pylint'
        bash .travis/checks/check-pylint.sh
    fold_end $?
    ####################

    ####################
    fold_start 'Check:copyright'
        bash .travis/checks/check-copyright.sh
    fold_end $?
    ####################

    ####################
    fold_start 'Check:signoff'
        python3 .travis/checks/check-signoff.py
    fold_end $?
    ####################
fi

exit $STATUS_CODE
