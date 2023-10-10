package core

import (
	"regexp"
	"strings"

	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/toolchain"
	"github.com/ARM-software/bob-build/core/toolchain/mapper"
	"github.com/google/blueprint"
)

type ModuleToolchainProps struct {
	// Flags that will be used for C and C++ compiles.
	Cflags []string

	// Flags that will be used for C compiles.
	Conlyflags []string

	// Flags that will be used for C++ compiles.
	Cppflags []string

	// Flags that will be used for .S compiles.
	Asflags []string

	// Flags that will be used for all link steps.
	Ldflags []string

	// Wrapper for all build commands (object file compilation *and* linking)
	Build_wrapper *string
}

type ToolchainFlagsProps struct {
	// `ModuleToolchain` module.
	Toolchain *string
}

// Strict targets will not support defaults by design.
//
// With this in mind, we will need a way to propagate
// common toolchain flags to targets (optimization etc).
type ModuleToolchain struct {
	module.ModuleBase

	SplittableProps

	Properties struct {
		ModuleToolchainProps
		StripProps
		TagableProps

		Target     TargetSpecific
		Host       TargetSpecific
		TargetType toolchain.TgtType `blueprint:"mutated"`

		// Arm Memory Tagging Extension
		AndroidMTEProps

		Features
	}
}

type BackendConfiguration interface {
	stripable
	GetBuildWrapperAndDeps(blueprint.ModuleContext) (string, []string)
}

// This interface provides configuration features
type BackendConfigurationProvider interface {
	GetBackendConfiguration(blueprint.ModuleContext) BackendConfiguration
}

func GetModuleBackendConfiguration(ctx blueprint.ModuleContext, m interface{}) BackendConfiguration {
	if capable, ok := m.(BackendConfigurationProvider); ok {
		if bc := capable.GetBackendConfiguration(ctx); bc != nil {
			return bc
		}
	}
	return nil
}

type ModuleToolchainInterface interface {
	Featurable
	targetSpecificLibrary
	flag.Provider
	Tagable
}

var _ ModuleToolchainInterface = (*ModuleToolchain)(nil)
var _ stripable = (*ModuleToolchain)(nil)
var _ BackendConfiguration = (*ModuleToolchain)(nil)

func (m *ModuleToolchain) FeaturableProperties() []interface{} {
	return []interface{}{
		&m.Properties.ModuleToolchainProps,
		&m.Properties.StripProps,
		&m.Properties.TagableProps,
	}
}

func (m *ModuleToolchain) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleToolchain) GenerateBuildActions(ctx blueprint.ModuleContext) {
	// `ModuleToolchain` does not generate any actions.
	// It only provides flags to be consumed by other modules.
}

func (m *ModuleToolchain) supportedVariants() []toolchain.TgtType {
	return []toolchain.TgtType{toolchain.TgtTypeHost, toolchain.TgtTypeTarget}
}

func (m *ModuleToolchain) disable() {
	// always enabled
}

func (m *ModuleToolchain) setVariant(tgt toolchain.TgtType) {
	m.Properties.TargetType = tgt
}

func (m *ModuleToolchain) getTarget() toolchain.TgtType {
	return m.Properties.TargetType
}

func (m *ModuleToolchain) getSplittableProps() *SplittableProps {
	return &m.SplittableProps
}

func (m *ModuleToolchain) getTargetSpecific(tgt toolchain.TgtType) *TargetSpecific {
	if tgt == toolchain.TgtTypeHost {
		return &m.Properties.Host
	} else if tgt == toolchain.TgtTypeTarget {
		return &m.Properties.Target
	}

	return nil
}

// Get the set of the module main properties for
// that target specific properties would be applied to
func (m *ModuleToolchain) targetableProperties() []interface{} {
	return []interface{}{
		&m.Properties.ModuleToolchainProps,
		&m.Properties.StripProps,
		&m.Properties.TagableProps,
	}
}

func (m *ModuleToolchain) GetBuildWrapperAndDeps(ctx blueprint.ModuleContext) (string, []string) {
	// Copies the behaviour from core/build.go
	if m.Properties.Build_wrapper != nil {
		depargs := map[string]string{}
		files, _ := getDependentArgsAndFiles(ctx, depargs)

		// Replace any property usage in buildWrapper
		buildWrapper := *m.Properties.Build_wrapper
		for k, v := range depargs {
			buildWrapper = strings.Replace(buildWrapper, "${"+k+"}", v, -1)
		}

		return buildWrapper, files
	}
	return "", []string{}
}

func (m *ModuleToolchain) FlagsOut() flag.Flags {
	lut := flag.FlagParserTable{
		{
			PropertyName: "Cflags",
			Tag:          flag.TypeCC | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Conlyflags",
			Tag:          flag.TypeC | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Cppflags",
			Tag:          flag.TypeCpp | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Asflags",
			Tag:          flag.TypeAsm | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
		{
			PropertyName: "Ldflags",
			Tag:          flag.TypeLinker | flag.TypeExported,
			Factory:      flag.FromStringOwned,
		},
	}

	return flag.ParseFromProperties(nil, lut, m.Properties)
}

func (m *ModuleToolchain) getDebugInfo() *string {
	return m.Properties.getDebugInfo()
}

func (m *ModuleToolchain) getDebugPath() *string {
	return m.Properties.getDebugPath()
}

func (m *ModuleToolchain) setDebugPath(path *string) {
	m.Properties.setDebugPath(path)
}

func (m *ModuleToolchain) stripOutputDir(g generatorBackend) string {
	return getBackendPathInBuildDir(g, string(m.Properties.TargetType), "strip")
}

func (m *ModuleToolchain) strip() bool {
	return m.Properties.Strip != nil && *m.Properties.Strip
}

func (m *ModuleToolchain) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleToolchain) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleToolchain) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleToolchain) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

func (m *ModuleToolchain) processPaths(ctx blueprint.BaseModuleContext) {
	if m.Properties.Build_wrapper != nil {
		// Copies core/build_props.go to duplicate the behaviour for `build_wrapper`
		*m.Properties.Build_wrapper = strings.TrimSpace(*m.Properties.Build_wrapper)
		firstWord := strings.SplitN(*m.Properties.Build_wrapper, " ", 1)[0]
		if firstWord[0] != '/' {
			if strings.ContainsAny(firstWord, "/") {
				*m.Properties.Build_wrapper = getBackendPathInSourceDir(getGenerator(ctx), *m.Properties.Build_wrapper)
			}
		}
	}
}

func ModuleToolchainFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleToolchain{}

	module.Properties.Features.Init(&config.Properties, ModuleToolchainProps{}, StripProps{}, TagableProps{})
	module.Properties.Host.init(&config.Properties, ModuleToolchainProps{}, StripProps{}, TagableProps{})
	module.Properties.Target.init(&config.Properties, ModuleToolchainProps{}, StripProps{}, TagableProps{})

	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}

var ToolchainModuleMap = mapper.New() // Global lookup for toolchain names

func RegisterToolchainModules(ctx blueprint.EarlyMutatorContext) {
	if _, ok := ctx.Module().(*ModuleToolchain); ok {
		ToolchainModuleMap.Add(ctx.ModuleDir(), ctx.ModuleName())
	}
}
