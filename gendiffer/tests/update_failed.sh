#!/bin/sh
# Updates all failed snapshots.
# Note, there is no way to currently parallize this as each invocation will lock the Bazel workspace.
bazelisk test //gendiffer/... |
	grep '//gendiffer/tests.*FAILED' |
	sed 's/\s.*$//' |
	while read target; do UPDATE_SNAPSHOTS=true bazelisk run $target; done
