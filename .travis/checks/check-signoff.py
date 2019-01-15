#!/bin/python3

import os
import sys
import json
import subprocess
#import urllib.request

status_code=0
pull_request_number=28
branch=''
commit_range="origin/master.."
pr_sha=''

if os.environ.get('CI') and os.environ.get('TRAVIS'):
	pull_request_number=os.environ['TRAVIS_PULL_REQUEST']
	branch=os.environ['TRAVIS_BRANCH']
	pr_sha=os.environ['TRAVIS_PULL_REQUEST_SHA']
	commit_range="origin/master..{}".format(pr_sha)


print("TRAVIS_PULL_REQUEST={}".format(pull_request_number))
print("TRAVIS_BRANCH={}".format(branch))
print("commit_range={}".format(commit_range))

if pull_request_number == False:
	sys.exit(0) # check for signoff in pull request ONLY

#https://api.github.com/repos/ARM-software/bob-build/pulls/28/commits
#/repos/:owner/:repo/pulls/:number/commits

#url = 'https://api.github.com/repos/ARM-software/bob-build/pulls/{}/commits?access_token={}'.format(pull_request_number, github_access_token)
#r = urllib.request.urlopen(url)
#data = json.loads(r.read().decode(r.info().get_param('charset') or 'utf-8'))

# For Travis variables check: https://docs.travis-ci.com/user/environment-variables/
# TRAVIS_COMMIT_RANGE - when force update it will contain previous old commit sha eg. 4b7d3xxxxxx...cf1b4xxxxxx
# cf1b4xxxxxx - forced pushed commit (after ammend)
# 4b7d3xxxxxx - previous commit
# so commands like git log wil fail in such cases
pretty='^%^{"sha":"%H","msg":"%B"}'
git_log=subprocess.check_output(['git', 'log', '--pretty={}'.format(pretty), commit_range]).decode("utf-8")
git_log=git_log.split('^%^')
for commit in git_log:
	commit=commit.strip().encode("unicode_escape").decode("utf-8")
	if len(commit) == 0:
		continue
	commit=json.loads(commit)
	if "Signed-off-by" not in commit['msg']:
		print("Commit: {} isn't Signed-off-by".format(commit['sha']))
		status_code=1
# for commit in data:
# 	if "Signed-off-by" not in commit['commit']['message']:
# 		print("Commit: {} isn't Signed-off-by".format(commit['sha']))
# 		print("Check: {}".format(commit['url']))
# 		status_code=1

sys.exit(status_code)
