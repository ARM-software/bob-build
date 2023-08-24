package core

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_basic_tag_operations(t *testing.T) {
	tags := TagableProps{Tags: []string{"owner:foo", "manual"}}
	assert.Equal(t, tags.GetTags(), []string{"owner:foo", "manual"})
	assert.True(t, tags.HasTag("manual"))
	assert.False(t, tags.HasTag("bar"))
}

func Test_regex_tag_ops(t *testing.T) {
	r, _ := regexp.Compile("^owner:.*")
	tags := TagableProps{Tags: []string{"owner:foo", "manual"}}
	assert.True(t, tags.HasTagRegex(r))
	assert.Equal(t, tags.GetTagsRegex(r), []string{"owner:foo"})
}
