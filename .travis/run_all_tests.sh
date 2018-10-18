#!/bin/bash
STATUS_CODE=0

fold_start() {
	travis_fold start "$1"
	travis_time_start
}

fold_end() {
	travis_time_finish
	travis_fold end "$1"
}

####################
fold_start 'vet'
go vet ./...
RESULT=$?
if [ $RESULT -ne 0 ]; then
	echo "Check 'go vet': FAIL"
	STATUS_CODE=1
else
	echo "Check 'go vet': OK"
fi
fold_end 'vet'

####################
fold_start 'gofmt'
bash ${BOB_ROOT}/.travis/validate_format.sh
RESULT=$?
if [ $RESULT -ne 0 ]; then
	echo "Check format: FAIL"
	STATUS_CODE=1
else
	echo "Check format: OK"
fi
fold_end 'gofmt'

####################
fold_start 'run_build_tests.sh'
bash ${BOB_ROOT}/.travis/run_build_tests.sh
RESULT=$?
if [ $RESULT -ne 0 ]; then
	echo "Check run_build_tests: FAIL"
	STATUS_CODE=1
else
	echo "Check run_build_tests: OK"
fi
fold_end 'run_build_tests.sh'

####################
fold_start 'run_go_tests.sh'
bash ${BOB_ROOT}/.travis/run_go_tests.sh
RESULT=$?
if [ $RESULT -ne 0 ]; then
	echo "Check run_go_tests: FAIL"
	STATUS_CODE=1
else
	echo "Check run_go_tests: OK"
fi
fold_end 'run_go_tests.sh'
exit $STATUS_CODE
