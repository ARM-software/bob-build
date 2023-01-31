module github.com/ARM-software/bob-build

go 1.18

require github.com/stretchr/testify v1.6.0

require github.com/google/blueprint v0.0.0-20200402195805-6957a46d38c9

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Note that currently this will print an error when running :gazelle-update-repos:
// gazelle: go_repository does not support file path replacements for github.com/google/blueprint -> ./blueprint
// However it does not stop the plugin from working and we need this to support non-Bazel builds.
replace github.com/google/blueprint v0.0.0-20200402195805-6957a46d38c9 => ./blueprint
