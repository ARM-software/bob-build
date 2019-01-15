#!/bin/python3

import os
import sys
import subprocess

class ccolors:
    GREEN = '\033[32;1m'
    WARNING = '\033[33;1m'
    FAIL = '\033[31;1m'
    ENDC = '\033[0m'

status_code=0

commit_range="origin/master.."

if os.environ.get('CI') and os.environ.get('TRAVIS'):
	commit_range = os.environ['TRAVIS_COMMIT_RANGE']

print("checking commit_range={}".format(commit_range))

def isAutomaticMerge(sha):
	'''Check whenever commit is nonconflicting merge. Such merge doesn't have any changes in it'''
	commit = subprocess.check_output(['git', 'show','--format=', '--raw', sha]).decode("utf-8")
	return len(commit) == 0

def isSignedOff(message):
	for line in message.splitlines():
		if line.startswith('Signed-off-by'):
			return True
	return False

try:
	# For Travis variables check: https://docs.travis-ci.com/user/environment-variables/
	# TRAVIS_COMMIT_RANGE - when force update it will contain previous old commit sha eg. 4b7d3xxxxxx...cf1b4xxxxxx
	# cf1b4xxxxxx - forced pushed commit (after amend)
	# 4b7d3xxxxxx - previous commit
	# so commands like git log wil fail in such cases, because 4b7d3xxxxxx doesn't exist in clean copy of repository
	git_shas = subprocess.check_output(['git', 'log', '--pretty=%H', commit_range]).decode("utf-8")
except subprocess.SubprocessError as err:
	print(ccolors.WARNING + "Can't find commits in range {}, so falling back to origin/master..".format(commit_range) + ccolors.ENDC)
	subprocess.check_output(['git', 'fetch', 'origin', 'master'])
	commit_range="FETCH_HEAD.."
	print(ccolors.WARNING + "[Failsafe] checking commit_range={}".format(commit_range) + ccolors.ENDC)
	git_shas = subprocess.check_output(['git', 'log', '--pretty=%H', commit_range]).decode("utf-8")

# Check all new/modified shas
git_shas = subprocess.check_output(['git', 'log', '--pretty=%H', commit_range]).decode("utf-8")
git_shas = git_shas.split('\n')
for sha in git_shas:
	sha = sha.strip()
	if len(sha) == 0:
		continue
	# Every change needs to be signed-off
	# Don't check automatically-generated merge commits, because these have no code to sign off anyway.
	# If there were conflicts, the merge commit will contain a diff of the resolutions, which does need signing-off.
	if isAutomaticMerge(sha):
		continue # skip such commit

	# Extract every commit message one by one. Maybe in commit someone placed some code or other odd chars.
	message = subprocess.check_output(['git', 'log', '--pretty=%B', '-n1', sha]).decode("utf-8")
	if not isSignedOff(message):
		print(ccolors.FAIL + "Commit: {} isn't Signed-off-by".format(sha) + ccolors.ENDC)
		status_code=1
	print(ccolors.GREEN + "Commit: {} is ok".format(sha) + ccolors.ENDC)

sys.exit(status_code)
