#!/usr/bin/env bash

BOB_ROOT=$(dirname ${0})/../..
cd "${BOB_ROOT}"
COPYRIGHT="copyright\s+[0-9,\s-]+"
CHECKED_COMMIT=${TRAVIS_COMMIT:-HEAD}
PARENT=$(git merge-base origin/master ${CHECKED_COMMIT})

exit=0
# Skip copyright verification when commit has revert in the title
for commit in $(git log --pretty="%H"  ${PARENT}..${CHECKED_COMMIT}); do
    title=$(git log --oneline -1 ${commit})
    echo $title
    if echo $title | grep -iq "revert"; then
        continue
    fi
    for file in $(git diff --name-only --diff-filter=ACM ${commit}^ ${commit}); do
        date_str=$(git show "${commit}:${file}" | head -15 | grep -iE "${COPYRIGHT}")
        if [[ "${date_str}" == "" ]]; then
            continue
        fi
        if [[ "${date_str}" != *"$(git show -s --format=%cd --date=format:%Y ${commit})"* ]]; then
            echo " ${file} is missing year update of copyright"
            exit+=1
        fi
    done
done

exit $exit
