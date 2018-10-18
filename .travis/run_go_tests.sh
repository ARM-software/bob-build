#!/bin/bash
set -eE
trap "echo '<------------- run_go_tests.sh failed'" ERR

cd ${BOB_WORKSPACE}/src/github.com/ARM-software/bob-build/
NAMESPACE="github.com/ARM-software/bob-build"
go test "$NAMESPACE/core" "$NAMESPACE/graph" "$NAMESPACE/utils"
#go test ./... # This should run all tests in current directory and all of its subdirectories

# go test -race -short ./...
