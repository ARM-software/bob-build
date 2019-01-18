#!/bin/bash

# Travis doesn't support multiple languages. So to support multiple Python
# versions, use an environment variable to select the version to be used, then
# create a wrapper script in a new PATH directory which invokes the correct
# version.
if [[ -n $PYTHON_SUFFIX ]]; then
    PYTHON_WRAPPER=~/pythonbin/python
    rm -f "$PYTHON_WRAPPER"
    mkdir -p $(dirname "$PYTHON_WRAPPER" ~/pythonbin)

    echo "#!/usr/bin/env bash" > "$PYTHON_WRAPPER"
    echo "python$PYTHON_SUFFIX" '"$@"' >> "$PYTHON_WRAPPER"
    chmod +x $PYTHON_WRAPPER

    export PATH=~/pythonbin:$PATH
    PYVER=$(python -c 'import sys; print("%d.%d." % (sys.version_info.major, sys.version_info.minor))') || STATUS_CODE=1
    echo "Check python version"
    if [[ $PYVER != "$PYTHON_SUFFIX."* ]]; then
        echo "Error: Python binary suffix is $PYTHON_SUFFIX, but version reported is $PYVER"
        echo "Set up Python version: FAIL"
        return 1
    fi
else
    echo "Error: No Python version selected"
    return 1
fi

return 0
