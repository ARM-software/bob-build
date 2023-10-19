package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsChildFilepath(t *testing.T) {
	child, err := IsChildFilepath("dir/", "dir/subdir")
	assert.True(t, child)
	assert.Nil(t, err)

	child, err = IsChildFilepath("other/", "dir/subdir")
	assert.False(t, child)
	assert.Nil(t, err)

	child, err = IsChildFilepath("dir/", "dir/dir/dir/dir/file")
	assert.True(t, child)
	assert.Nil(t, err)
}
