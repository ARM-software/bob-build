# Gazelle test for checking the order of generated rules

Test checks that generated rules order conforms to bob modules
defined in `build.bp` files.

## Update the output

To automatically update the expected output, use `UPDATE_SNAPSHOT=true`, eg:

```sh
UPDATE_SNAPSHOT=true bazelisk test //...
```
