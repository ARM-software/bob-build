#!/usr/bin/env python

from __future__ import print_function

import os
import sys
import subprocess

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

    commit_range = "origin/master.."

    if os.environ.get('CI') and os.environ.get('TRAVIS'):
        commit_range = os.environ['TRAVIS_COMMIT_RANGE']

    print("checking commit_range={}".format(commit_range))

    try:
        # For Travis variables check: https://docs.travis-ci.com/user/environment-variables/

        # TRAVIS_COMMIT_RANGE - when force update it will contain previous
        #                       old commit sha eg. 4b7d3xxxxxx...cf1b4xxxxxx
        # cf1b4xxxxxx - forced pushed commit (after amend)
        # 4b7d3xxxxxx - previous commit
        #
        # so commands like git log will fail in such cases, because
        # 4b7d3xxxxxx doesn't exist in clean copy of repository
        cmd = ['git', 'log', '--pretty=%H', commit_range]
        git_shas = subprocess.check_output(cmd).decode("utf-8")

    except subprocess.SubprocessError:
        print(WARNING +
              "Can't find commits in range {}, so falling back to origin/master..".
              format(commit_range) + ENDC)
        cmd = ['git', 'fetch', 'origin', 'master']
        subprocess.check_output(cmd)
        commit_range = "FETCH_HEAD.."
        print(WARNING +
              "[Failsafe] checking commit_range={}".
              format(commit_range) + ENDC)
        cmd = ['git', 'log', '--pretty=%H', commit_range]
        git_shas = subprocess.check_output(cmd).decode("utf-8")

    # Check all new/modified shas
    cmd = ['git', 'log', '--pretty=%H', commit_range]
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
