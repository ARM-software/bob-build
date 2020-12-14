#!/usr/bin/env bash
set -eE
trap "echo '<------------- $(basename ${0}) failed'" ERR

NAMESPACE="github.com/ARM-software/bob-build"
go test "$NAMESPACE/core" "$NAMESPACE/internal/escape" "$NAMESPACE/internal/graph" "$NAMESPACE/internal/utils"
#go test ./... # This should run all tests in current directory and all of its subdirectories

# go test -race -short ./...
