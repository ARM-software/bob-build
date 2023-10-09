package core

import (
	"path/filepath"
	"regexp"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// TransformSourceProps contains the properties allowed in the
// bob_transform_source module. This module supports one command execution
// per input file.
type TransformSourceProps struct {
	// The regular expression that is used to transform the source path to the target path.
	Out struct {
		// Regular expression to capture groups from srcs
		Match string
		// Names of outputs, which can use capture groups from match
		Replace []string
		// List of implicit sources. Implicit sources are input files that do not get mentioned on the command line,
		// and are not specified in the explicit sources.
		Implicit_srcs []string
	}

	// Stores the files generated
	ResolvedOut file.Paths `blueprint:"mutated"`
}

func (tsp *TransformSourceProps) inoutForSrc(re *regexp.Regexp, source file.Path, depfile *bool, rspfile bool) (io inout) {
	io.in = []string{source.BuildPath()}

	for _, rep := range tsp.Out.Replace {
		// TODO: figure out the outs here.
		out := filepath.Join(re.ReplaceAllString(source.ScopedPath(), rep))
		io.out = append(io.out, out)
	}

	if proptools.Bool(depfile) {
		io.depfile = getDepfileName(source.UnScopedPath())
	}

	for _, implSrc := range tsp.Out.Implicit_srcs {
		implSrc = re.ReplaceAllString(source.UnScopedPath(), implSrc)
		io.implicitSrcs = append(io.implicitSrcs, source.BuildPath())
	}

	if rspfile {
		io.rspfile = getRspfileName(source.UnScopedPath())
	}

	return
}

// The module that can generate sources using a multiple execution
// The command will be run once per src file- with $in being the path in "srcs" and $out being the path transformed
// through the regexp defined by out.match and out.replace. The regular expression that is used is
// in regexp.compiled(out.Match).ReplaceAllString(src[i], out.Replace). See https://golang.org/pkg/regexp/ for more
// information.
// The working directory will be the source directory, and all paths will be relative to the source directory
// if not else noted
type ModuleTransformSource struct {
	ModuleGenerateCommon
	Properties struct {
		TransformSourceProps
	}
}

// All interfaces supported by filegroup
type transformSourceInterface interface {
	installable
	// file.DynamicProvider clashes with `installable` on older Go versions
	ResolveOutFiles(blueprint.BaseModuleContext)
	file.Consumer
	file.Resolver
}

var _ transformSourceInterface = (*ModuleTransformSource)(nil) // impl check

func (m *ModuleTransformSource) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool {
			// TODO: Consider adding a better group tag
			return f.IsNotType(file.TypeRsp) &&
				f.IsNotType(file.TypeDep)
		},
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleTransformSource) implicitOutputs() []string {
	return file.GetImplicitOutputs(m)
}

func (m *ModuleTransformSource) FeaturableProperties() []interface{} {
	return append(m.ModuleGenerateCommon.FeaturableProperties(), &m.Properties.TransformSourceProps)
}

func (m *ModuleTransformSource) sourceInfo(ctx blueprint.ModuleContext, g generatorBackend) []file.Path {
	return m.GetFiles(ctx)
}

func (m *ModuleTransformSource) ResolveFiles(ctx blueprint.BaseModuleContext) {
	m.getLegacySourceProperties().ResolveFiles(ctx)
}

func (m *ModuleTransformSource) GetFiles(ctx blueprint.BaseModuleContext) file.Paths {
	return m.getLegacySourceProperties().GetFiles(ctx)
}

func (m *ModuleTransformSource) GetDirectFiles() file.Paths {
	return m.getLegacySourceProperties().GetDirectFiles()
}

func (m *ModuleTransformSource) GetTargets() []string {
	gc, _ := getGenerateCommon(m)
	return gc.Properties.Generated_sources
}

func (m *ModuleTransformSource) OutFiles() file.Paths {
	gc, _ := getGenerateCommon(m)
	return append(m.Properties.ResolvedOut, gc.OutFiles()...)
}

func (m *ModuleTransformSource) OutFileTargets() []string {
	return []string{}
}

func (m *ModuleTransformSource) FlagsOut() (flags flag.Flags) {
	gc, _ := getGenerateCommon(m)
	for _, str := range gc.Properties.Export_gen_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

func (m *ModuleTransformSource) ResolveOutFiles(ctx blueprint.BaseModuleContext) {
	re := regexp.MustCompile(m.Properties.Out.Match)

	// TODO: Refactor this to share code with generateInouts, right now the ctx type is different so no sharing is possible.

	m.GetFiles(ctx).ForEach(
		func(fp file.Path) bool {
			io := m.Properties.inoutForSrc(re, fp, m.ModuleGenerateCommon.Properties.Depfile,
				m.ModuleGenerateCommon.Properties.Rsp_content != nil)
			for _, out := range io.out {
				fp := file.NewPath(out, ctx.ModuleName(), file.TypeGenerated|file.TypeInstallable)
				m.Properties.ResolvedOut = m.Properties.ResolvedOut.AppendIfUnique(fp)
			}
			return true
		})
}

// Return an inouts structure naming all the files associated with
// each transformSource input.
//
// The inputs are full paths (possibly using build system variables).
//
// The outputs are relative to the output directory. This applies
// to out, depfile and rspfile. The output directory (if needed) needs to be
// added in by the backend specific GenerateBuildAction()
func (m *ModuleTransformSource) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	var inouts []inout
	re := regexp.MustCompile(m.Properties.Out.Match)

	for _, source := range m.sourceInfo(ctx, g) {
		io := m.Properties.inoutForSrc(re, source, m.ModuleGenerateCommon.Properties.Depfile,
			m.ModuleGenerateCommon.Properties.Rsp_content != nil)
		inouts = append(inouts, io)
	}

	return inouts
}

func (m *ModuleTransformSource) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).transformSourceActions(m, ctx)
	}
}

func (m ModuleTransformSource) GetProperties() interface{} {
	return m.Properties
}

func transformSourceFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleTransformSource{}
	module.ModuleGenerateCommon.init(&config.Properties,
		GenerateProps{}, TransformSourceProps{})

	return module, []interface{}{&module.ModuleGenerateCommon.Properties,
		&module.Properties,
		&module.SimpleName.Properties}
}
