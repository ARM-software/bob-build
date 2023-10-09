package core

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/module"
	"github.com/ARM-software/bob-build/core/tag"
	"github.com/ARM-software/bob-build/internal/utils"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type KernelProps struct {
	// Linux kernel config options to emulate. These are passed to Kbuild in
	// the 'make' command-line, and set in the source code via EXTRA_CFLAGS
	Kbuild_options []string
	// Kernel modules which this module depends on
	Extra_symbols []string
	// Arguments to pass to kernel make invocation
	Make_args []string
	// Kernel directory location
	Kernel_dir *string
	// Compiler prefix for kernel build
	Kernel_cross_compile *string
	// Kernel target compiler
	Kernel_cc *string
	// Kernel host compiler
	Kernel_hostcc *string
	// Kernel linker
	Kernel_ld *string
	// Target triple when using clang as the compiler
	Kernel_clang_triple *string
}

func (k *KernelProps) processPaths(ctx blueprint.BaseModuleContext) {
	prefix := projectModuleDir(ctx)

	// join module dir with relative kernel dir
	kdir := proptools.String(k.Kernel_dir)
	if kdir != "" && !filepath.IsAbs(kdir) {
		kdir = filepath.Join(prefix, kdir)
		k.Kernel_dir = proptools.StringPtr(kdir)
	}
}

type ModuleKernelObject struct {
	module.ModuleBase
	Properties struct {
		Features
		CommonProps
		KernelProps
		TagableProps
		Defaults []string
	}
}

// kernelModule supports the following functionality:
type kernelModuleInterface interface {
	defaultable // sharing properties via defaults
	Featurable  // feature-specific properties
	installable // installation
	enableable  // module enabling/disabling
	aliasable   // appending to aliases
	pathProcessor
	file.Resolver
	Tagable
}

var _ kernelModuleInterface = (*ModuleKernelObject)(nil) // impl check

func (m *ModuleKernelObject) OutFiles() file.Paths {
	return file.Paths{file.NewPath(m.outputName()+".ko", m.Name(), file.TypeKernelModule|file.TypeInstallable)}
}
func (m *ModuleKernelObject) OutFileTargets() []string {
	return []string{}
}

func (m *ModuleKernelObject) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.Properties.ResolveFiles(ctx)
}

func (m *ModuleKernelObject) defaults() []string {
	return m.Properties.Defaults
}

func (m *ModuleKernelObject) defaultableProperties() []interface{} {
	return []interface{}{&m.Properties.CommonProps, &m.Properties.KernelProps}
}

func (m *ModuleKernelObject) FeaturableProperties() []interface{} {
	return []interface{}{&m.Properties.CommonProps, &m.Properties.KernelProps}
}

func (m *ModuleKernelObject) Features() *Features {
	return &m.Properties.Features
}

func (m *ModuleKernelObject) outputName() string {
	return m.Name()
}

func (m *ModuleKernelObject) altName() string {
	return m.outputName()
}

func (m *ModuleKernelObject) altShortName() string {
	return m.altName()
}

func (m *ModuleKernelObject) shortName() string {
	return m.Name()
}

func (m *ModuleKernelObject) getEnableableProps() *EnableableProps {
	return &m.Properties.EnableableProps
}

func (m *ModuleKernelObject) getAliasList() []string {
	return m.Properties.getAliasList()
}

func (m *ModuleKernelObject) getInstallableProps() *InstallableProps {
	return &m.Properties.InstallableProps
}

func (m *ModuleKernelObject) getInstallDepPhonyNames(ctx blueprint.ModuleContext) []string {
	return getShortNamesForDirectDepsWithTags(ctx, tag.InstallTag, tag.KernelModuleTag)
}

func (m *ModuleKernelObject) processPaths(ctx blueprint.BaseModuleContext) {
	m.Properties.CommonProps.processPaths(ctx)
	m.Properties.KernelProps.processPaths(ctx)
}

func (m *ModuleKernelObject) extraSymbolsModules(ctx blueprint.BaseModuleContext) (modules []*ModuleKernelObject) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.KernelModuleTag },
		func(m blueprint.Module) {
			if km, ok := m.(*ModuleKernelObject); ok {
				modules = append(modules, km)
			} else {
				utils.Die("invalid extra_symbols, %s not a kernel module", ctx.OtherModuleName(m))
			}
		})

	return
}

func (m *ModuleKernelObject) extraSymbolsFiles(ctx blueprint.BaseModuleContext) (files []string) {
	ctx.VisitDirectDepsIf(
		func(m blueprint.Module) bool { return ctx.OtherModuleDependencyTag(m) == tag.KernelModuleTag },
		func(m blueprint.Module) {
			path := filepath.Join(backend.Get().KernelModOutputDir(), m.Name(), "Module.symvers")
			files = append(files, path)
		})
	return
}

func (m *ModuleKernelObject) HasTagRegex(query *regexp.Regexp) bool {
	return m.Properties.TagableProps.HasTagRegex(query)
}

func (m *ModuleKernelObject) HasTag(query string) bool {
	return m.Properties.TagableProps.HasTag(query)
}

func (m *ModuleKernelObject) GetTagsRegex(query *regexp.Regexp) []string {
	return m.Properties.TagableProps.GetTagsRegex(query)
}

func (m *ModuleKernelObject) GetTags() []string {
	return m.Properties.TagableProps.GetTags()
}

type kbuildArgs struct {
	KmodBuild          string
	ExtraIncludes      string
	ExtraCflags        string
	KernelDir          string
	KernelCrossCompile string
	KbuildOptions      string
	MakeArgs           string
	OutputModuleDir    string
	CCFlag             string
	HostCCFlag         string
	ClangTripleFlag    string
	LDFlag             string
}

func (a kbuildArgs) toDict() map[string]string {
	return map[string]string{
		"kmod_build":           a.KmodBuild,
		"extra_includes":       a.ExtraIncludes,
		"extra_cflags":         a.ExtraCflags,
		"kernel_dir":           a.KernelDir,
		"kernel_cross_compile": a.KernelCrossCompile,
		"kbuild_options":       a.KbuildOptions,
		"make_args":            a.MakeArgs,
		"output_module_dir":    a.OutputModuleDir,
		"cc_flag":              a.CCFlag,
		"hostcc_flag":          a.HostCCFlag,
		"clang_triple_flag":    a.ClangTripleFlag,
		"ld_flag":              a.LDFlag,
	}
}

func (m *ModuleKernelObject) generateKbuildArgs(ctx blueprint.BaseModuleContext) kbuildArgs {
	var extraIncludePaths []string

	extraCflags := m.Properties.Cflags

	for _, includeDir := range m.Properties.IncludeDirsProps.Local_include_dirs {
		includeDir = "-I" + getBackendPathInSourceDir(getGenerator(ctx), includeDir)
		extraIncludePaths = append(extraIncludePaths, includeDir)
	}

	for _, includeDir := range m.Properties.IncludeDirsProps.Include_dirs {
		includeDir = "-I" + includeDir
		extraIncludePaths = append(extraIncludePaths, includeDir)
	}

	kmodBuild := getBackendPathInBobScriptsDir(getGenerator(ctx), "kmod_build.py")
	kdir := proptools.String(m.Properties.KernelProps.Kernel_dir)
	if kdir != "" && !filepath.IsAbs(kdir) {
		kdir = getBackendPathInSourceDir(getGenerator(ctx), kdir)
	}

	kbuildOptions := ""
	if len(m.Properties.KernelProps.Kbuild_options) > 0 {
		kbuildOptions = "--kbuild-options " + strings.Join(m.Properties.KernelProps.Kbuild_options, " ")
	}

	hostToolchain := proptools.String(m.Properties.KernelProps.Kernel_hostcc)
	if hostToolchain != "" {
		hostToolchain = "--hostcc " + hostToolchain
	}

	kernelToolchain := proptools.String(m.Properties.KernelProps.Kernel_cc)
	if kernelToolchain != "" {
		kernelToolchain = "--cc " + kernelToolchain
	}

	clangTriple := proptools.String(m.Properties.KernelProps.Kernel_clang_triple)
	if clangTriple != "" {
		clangTriple = "--clang-triple " + clangTriple
	}

	ld := proptools.String(m.Properties.KernelProps.Kernel_ld)
	if ld != "" {
		ld = "--ld " + ld
	}

	return kbuildArgs{
		KmodBuild:          kmodBuild,
		ExtraIncludes:      strings.Join(extraIncludePaths, " "),
		ExtraCflags:        strings.Join(extraCflags, " "),
		KernelDir:          kdir,
		KernelCrossCompile: proptools.String(m.Properties.KernelProps.Kernel_cross_compile),
		KbuildOptions:      kbuildOptions,
		MakeArgs:           strings.Join(m.Properties.KernelProps.Make_args, " "),
		// The kernel module builder replicates the out-of-tree module's source tree structure.
		// The kernel module will be at its equivalent position in the output tree.
		OutputModuleDir: filepath.Join(backend.Get().KernelModOutputDir(), m.Name(), projectModuleDir(ctx)),
		CCFlag:          kernelToolchain,
		HostCCFlag:      hostToolchain,
		LDFlag:          ld,
		ClangTripleFlag: clangTriple,
	}
}

func (m *ModuleKernelObject) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).kernelModuleActions(m, ctx)
	}
}

func (m ModuleKernelObject) GetProperties() interface{} {
	return m.Properties
}

func kernelModuleFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleKernelObject{}

	module.Properties.Features.Init(&config.Properties, CommonProps{}, KernelProps{})

	return module, []interface{}{&module.Properties, &module.SimpleName.Properties}
}
