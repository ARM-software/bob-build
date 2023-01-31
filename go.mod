module github.com/ARM-software/bob-build

go 1.11

require (
	github.com/google/blueprint v0.0.0-20200402195805-6957a46d38c9
	github.com/stretchr/testify v1.6.0
)

replace github.com/google/blueprint v0.0.0-20200402195805-6957a46d38c9 => ./blueprint
