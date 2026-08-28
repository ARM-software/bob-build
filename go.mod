module github.com/ARM-software/bob-build

go 1.18

require (
	github.com/google/blueprint v0.0.0-20200402195805-6957a46d38c9
	github.com/stretchr/testify v1.6.0
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.1.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/google/blueprint v0.0.0-20200402195805-6957a46d38c9 => ./blueprint
