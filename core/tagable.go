package core

import (
	"regexp"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type TagableProps struct {
	Tags []string
}

type Tagable interface {
	blueprint.Module

	// Returns true if any of the tags match the expression
	HasTagRegex(*regexp.Regexp) bool

	// Return true if any of the tags match query
	HasTag(string) bool

	// Returns all tags matching regex
	GetTagsRegex(*regexp.Regexp) []string

	// Returns all tags
	GetTags() []string
}

func (p *TagableProps) HasTagRegex(query *regexp.Regexp) bool {
	for _, tag := range p.Tags {
		if query.MatchString(tag) {
			return true
		}
	}
	return false
}

func (p *TagableProps) HasTag(query string) bool {
	for _, tag := range p.Tags {
		if tag == query {
			return true
		}
	}

	return false
}

func (p *TagableProps) GetTagsRegex(query *regexp.Regexp) []string {
	return utils.Filter(func(s string) bool { return query.MatchString(s) }, p.Tags)
}

func (p *TagableProps) GetTags() []string { return p.Tags }
