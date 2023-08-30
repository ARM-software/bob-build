package core

import (
	"regexp"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"

	"github.com/google/blueprint"
)

type ModuleFilegroup struct {
	module.ModuleBase
	Properties struct {
		SourceProps
		TagableProps
		Features
	}
}

// All interfaces supported by filegroup
type filegroupInterface interface {
	pathProcessor
	file.Resolver
	file.Provider
	Tagable
}

var _ filegroupInterface = (*ModuleFilegroup)(nil) // impl check

func (m *ModuleFilegroup) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleFilegroup) OutFiles() file.Paths {
	return m.Properties.GetDirectFiles()
}

func (m *ModuleFilegroup) OutFileTargets() []string {
	return m.Properties.GetTargets()
}

func (m *ModuleFilegroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).filegroupActions(m, ctx)
}

func (m *ModuleFilegroup) shortName() string {
	return m.Name()
}

func (m *ModuleFilegroup) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.SourceProps.processPaths(ctx)
}

func (m *ModuleFilegroup) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.SourceProps,
		&m.Properties.TagableProps,
	}
}

func (m *ModuleFilegroup) Features() *Features {
	return &m.Properties.Features
}

func (m ModuleFilegroup) GetProperties() interface{} {
	return m.Properties
}

func (m *ModuleFilegroup) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleFilegroup) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleFilegroup) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleFilegroup) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

func filegroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleFilegroup{}
	module.Properties.Features.Init(&config.Properties, SourceProps{}, TagableProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
