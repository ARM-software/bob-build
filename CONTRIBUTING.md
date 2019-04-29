Contributing to Bob
===================

Contributions are welcomed! Please read the following to get started.

## License

Contributions to this project are accepted under the Apache 2.0 license.

Please also ensure that each commit in the series has at least one
`Signed-off-by:` line, using your real name and email address. The names in the
`Signed-off-by:` and `Author:` lines must match. If anyone else contributes to
the commit, they must also add their own `Signed-off-by:` line. By adding this
line the contributor certifies the contribution is made under the terms of the
[Developer Certificate of Origin (DCO)](DCO.txt).

## Making changes

### Coding Style

- Please use the `gofmt` and `bpfmt` tools when formatting Go and Blueprint
  files, respectively.
- For other languages (e.g. Python and Mconfig), please attempt to be
  consistent with the existing style.

### Testing

Bob has three kinds of tests:

- The `tests` directory, containing a collection of different modules which
  should all build, or be deliberately disabled. Please test this on Linux
  or Android.

  - Linux: (run inside Bob directory)

    ```bash
    cd tests
    ./bootstrap
    ./build/config
    ./build/buildme
    ```
    (thereafter just run `buildme`)

  - Android: (substitute variables appropriately - `$ANDROID_TOP` is a full
    checkout of the Android source code)

    ```bash
    # Usual Android setup - envsetup/lunch/etc
    mkdir -p $ANDROID_TOP/external/bob
    bindfs -n $BOB_LOCATION $ANDROID_TOP/external/bob
    cd $ANDROID_TOP/external/bob/tests
    ./bootstrap_android ANDROID=y
    mm
    ```
    (thereafter just run `mm`)

- Go unit tests, which can be run using `go test` after running
  `setup_workspace_for_bob.bash`:

  ```bash
  export GOPATH=~/go
  ./scripts/setup_workspace_for_bob.bash
  go test github.com/ARM-software/bob-build/core \
          github.com/ARM-software/bob-build/graph \
          github.com/ARM-software/bob-build/utils
  # OR:
  cd $GOPATH/src/github.com/ARM-software/bob-build/core
  go test
  # OR:
  cd $GOPATH/src/github.com/ARM-software/bob-build
  go test ./core ./graph ./utils
  ```

- The configuration system tests:

  ```bash
  ./config_system/tests/run_tests.py
  ./config_system/tests/run_tests_formatter.py
  pytest ./config_system
  ```

  These tests require the `pytest`, `pytest-catchlog`, `pytest-mock` and `mock`
  Python packages.

  Note: Do not run `pytest` in the top-level `bob-build` directory; it will
  fail during test discovery because of the recursive symlink inside the main
  Bob `tests` directory.

If your contribution is a bugfix, please consider adding a new test to prevent
future regressions.

### Submitting changes

- Create a pull request to the `master` branch.
- All submissions will require code review before merging.
- As mentioned above, please ensure your commit message contains a
  `Signed-off-by` tag.
