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

# Travis doesn't support multiple languages. So to support multiple
# Python versions, create a wrapper script in a new PATH directory
# which invokes the correct version.
#
# This function modifies PATH
set_python_version() {
    local SUFFIX=$1
    if [[ -n $SUFFIX ]]; then
        local PYTHON_WRAPPER=~/pythonbin/python
        rm -f "$PYTHON_WRAPPER"
        mkdir -p $(dirname "$PYTHON_WRAPPER" ~/pythonbin)

        echo "#!/usr/bin/env bash" > "$PYTHON_WRAPPER"
        echo "python$PYTHON_SUFFIX" '"$@"' >> "$PYTHON_WRAPPER"
        chmod +x $PYTHON_WRAPPER

        export PATH=~/pythonbin:$PATH
        local PYVER=$(python -c 'import sys; print("%d.%d." % (sys.version_info.major, sys.version_info.minor))')
        echo "Check python version"
        if [[ $PYVER != "$SUFFIX."* ]]; then
            echo "Error: Python binary suffix is $SUFFIX, but version reported is $PYVER"
            echo "Set up Python version: FAIL"
            return 1
        fi
    else
        echo "Error: No Python version selected"
        return 1
    fi

    return 0
}
