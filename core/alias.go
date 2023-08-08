package core

import (
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/tag"

	"github.com/google/blueprint"
)

// Modules implementing the aliasable interface can be referenced by a
// bob_alias module
type aliasable interface {
	getAliasList() []string
}

// AliasableProps are embedded in modules which can be aliased
type AliasableProps struct {
	// Adds this module to an alias
	Add_to_alias []string
}

func (p *AliasableProps) getAliasList() []string {
	return p.Add_to_alias
}

// AliasProps describes the properties of the bob_alias module
type AliasProps struct {
	// Modules that this alias will cause to build
	Srcs []string
	AliasableProps
}

// Type representing each bob_alias module
type ModuleAlias struct {
	module.ModuleBase
	Properties struct {
		AliasProps
		Features
	}
}

func (m *ModuleAlias) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleAlias) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.AliasProps}
}

func (m *ModuleAlias) getAliasList() []string {
	return m.Properties.getAliasList()
}

// Called by Blueprint to generate the rules associated with the alias.
// This is forwarded to the backend to handle.
func (m *ModuleAlias) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getGenerator(ctx).aliasActions(m, ctx)
}

func (m ModuleAlias) GetProperties() interface{} {
	return m.Properties
}

// Create the structure representing the bob_alias
func aliasFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleAlias{}
	module.Properties.Features.Init(&config.Properties, AliasProps{})
	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}

// Setup dependencies between aliases and their targets
func aliasMutator(ctx blueprint.BottomUpMutatorContext) {
	if a, ok := ctx.Module().(*ModuleAlias); ok {
		parseAndAddVariationDeps(ctx, tag.AliasTag, a.Properties.Srcs...)
	}
	if a, ok := ctx.Module().(aliasable); ok {
		for _, s := range a.getAliasList() {
			ctx.AddReverseDependency(ctx.Module(), tag.AliasTag, s)
		}
	}
}
