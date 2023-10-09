package core

import (
	"strings"

	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type ModuleSharedLibrary struct {
	ModuleLibrary
	fileNameExtension string
}

const (
	tocExt = ".toc"
)

// sharedLibrary supports:
// * producing output using the linker
// * producing a shared library
// * stripping symbols from output
var _ linkableModule = (*ModuleSharedLibrary)(nil)
var _ sharedLibProducer = (*ModuleSharedLibrary)(nil)
var _ stripable = (*ModuleSharedLibrary)(nil)
var _ libraryInterface = (*ModuleSharedLibrary)(nil) // impl check

func (m *ModuleSharedLibrary) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		func(f file.Path) bool { return !f.IsSymLink() && !f.IsType(file.TypeToc) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *ModuleSharedLibrary) OutFiles() (files file.Paths) {

	so := file.NewPath(m.getRealName(), string(m.getTarget()), file.TypeShared|file.TypeInstallable)
	files = append(files, so)

	toc := file.NewPath(m.getRealName()+tocExt, string(m.getTarget()), file.TypeToc)
	files = append(files, toc)

	if m.ModuleLibrary.Properties.Library_version != "" {
		soname := m.getSoname()
		realName := m.getRealName()
		if soname == realName {
			utils.Die("module %s has invalid library_version '%s'",
				m.Name(),
				m.ModuleLibrary.Properties.Library_version)
		}

		link1 := file.NewLink(soname, string(m.getTarget()), &so, file.TypeInstallable)
		link2 := file.NewLink(m.getLinkName(), string(m.getTarget()), &link1, file.TypeInstallable)
		files = append(files, link2, link1)
	}

	return
}

func (m *ModuleSharedLibrary) OutFileTargets() []string { return []string{} }

func (m *ModuleSharedLibrary) getLinkName() string {
	return m.outputName() + m.fileNameExtension
}

func (m *ModuleSharedLibrary) getSoname() string {
	name := m.getLinkName()
	if m.ModuleLibrary.Properties.Library_version != "" {
		var v = strings.Split(m.ModuleLibrary.Properties.Library_version, ".")
		name += "." + v[0]
	}
	return name
}

func (m *ModuleSharedLibrary) FlagsIn() flag.Flags {
	flags := flag.ParseFromProperties(nil, m.getFlagInLut(), m.Properties)

	if m.Properties.Library_version != "" {
		flags = append(flags, flag.FromString("-Wl,-soname,"+m.getSoname(), flag.TypeLinker))
	}

	return flags
}

func (m *ModuleSharedLibrary) getRealName() string {
	name := m.getLinkName()
	if m.ModuleLibrary.Properties.Library_version != "" {
		name += "." + m.ModuleLibrary.Properties.Library_version
	}
	return name
}

func (m *ModuleSharedLibrary) strip() bool {
	return m.Properties.Strip != nil && *m.Properties.Strip
}

func (m *ModuleSharedLibrary) GetStripable(ctx blueprint.ModuleContext) stripable {
	return m
}

func (m *ModuleSharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).sharedActions(m, ctx)
	}
}

func (m *ModuleSharedLibrary) outputFileName() string {
	// Since we link against libraries using the library flag style,
	// -lmod, return the name of the link library here rather than the
	// real, versioned library.
	return m.getLinkName()
}

func (m *ModuleSharedLibrary) getTocName() string {
	return m.getRealName() + tocExt
}

func (m ModuleSharedLibrary) GetProperties() interface{} {
	return m.ModuleLibrary.Properties
}

func sharedLibraryFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &ModuleSharedLibrary{}
	if config.Properties.GetBool("osx") {
		module.fileNameExtension = ".dylib"
	} else {
		module.fileNameExtension = ".so"
	}
	return module.LibraryFactory(config, module)
}
