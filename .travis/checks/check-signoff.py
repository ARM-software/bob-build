#!/usr/bin/env python

from __future__ import print_function

import os
import subprocess
import sys

GREEN = '\033[32;1m'
WARNING = '\033[33;1m'
FAIL = '\033[31;1m'
ENDC = '\033[0m'


def isAutomaticMerge(sha):
    """
    Check whether a commit is a non-conflicting merge. Such merge
    doesn't have any changes in it
    """
    cmd = ['git', 'show', '--format=', '--raw', sha]
    commit = subprocess.check_output(cmd).decode("utf-8")
    return len(commit) == 0


def isSignedOff(message):
    """
    Check whether a commit message contains Signed-off-by tag
    """
    for line in message.splitlines():
        if line.startswith('Signed-off-by'):
            return True
    return False


def main():
    """
    Main entry point for check-signoff script
    """
    status_code = 0

    # Check all new/modified shas
    cmd = ['git', 'log', '--pretty=%H', 'origin/master..HEAD']
    git_shas = subprocess.check_output(cmd).decode("utf-8")
    git_shas = git_shas.split('\n')
    for sha in git_shas:
        sha = sha.strip()
        if len(sha) == 0:
            continue

        # Every change needs to be signed-off
        #
        # Don't check automatically-generated merge commits, because these
        # have no code to sign off anyway.
        #
        # If there were conflicts, the merge commit will contain a diff of
        # the resolutions, which does need signing-off.
        if isAutomaticMerge(sha):
            continue

        # Extract every commit message one by one. Maybe in commit someone
        # placed some code or other odd chars.
        cmd = ['git', 'log', '--pretty=%B', '-n1', sha]
        message = subprocess.check_output(cmd).decode("utf-8")
        if not isSignedOff(message):
            print(FAIL + "Commit: {} isn't Signed-off-by".format(sha) + ENDC)
            status_code = 1
        else:
            print(GREEN + "Commit: {} is ok".format(sha) + ENDC)

        return status_code


if __name__ == "__main__":
    sys.exit(main())
