# Gazelle test for `filegroup` generation

Test checks the generated output based on defined `bob_filegroup` and `bob_glob` targets.

## Update the output

To automatically update the expected output, use `UPDATE_SNAPSHOT=true`, eg:

```sh
UPDATE_SNAPSHOT=true bazelisk test //...
```
