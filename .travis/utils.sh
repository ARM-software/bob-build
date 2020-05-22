FOLD=""
STATUS_CODE=0

fold_start() {
    FOLD="$1"
    travis_fold start "$FOLD"
    travis_time_start
    printf "\e[33;1m$FOLD\e[0m\n"
}

fold_end() {
    local RESULT=$1
    travis_time_finish
    travis_fold end "$FOLD"

    echo -n "$FOLD: "
    if [ $RESULT -eq 0 ]; then
        result_ok
    else
        STATUS_CODE=1
        result_fail
    fi
}

result_ok() {
    printf "\e[32;1mOK\e[0m\n"
}

result_skip() {
    local MSG=$1

    echo -n "$MSG "
    printf "\e[33;1mSKIP\e[0m\n"
}

result_fail() {
    printf "\e[31;1mFAIL\e[0m\n"
}
