#!/usr/bin/env bash
set -eE
trap 'echo "<------------- "$(basename ${0})" failed"' ERR

SCRIPT_DIR=$(dirname "${BASH_SOURCE[0]}")
BOB_ROOT="${SCRIPT_DIR}/.."

pushd "${BOB_ROOT}" &> /dev/null

build_dir=build-example
rm -rf "${build_dir}"

# Test example setup if it's buildable
cp -r example ${build_dir}
pushd ${build_dir} &> /dev/null
# Link to existing bob checkout
ln -s ../ bob-build
./bootstrap_linux.bash

# Generate example source file
cat > hello_world.cpp << EOF
int main()
{
    int hello = 1;
    hello += 1;
    return 0;
}
EOF

case "$(uname -s)" in
    Darwin*)
        OS=OSX
        ;;
    *)
        OS=LINUX
        ;;
esac

cd build
./config "$OS=y"
./buildme

# Return to BOB_ROOT
popd &> /dev/null

# Clean up, return to origin dir
rm -rf "${build_dir}"
popd &> /dev/null
