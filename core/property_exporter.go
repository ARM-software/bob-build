package core

type propertyExporter interface {
	exportCflags() []string
	exportIncludeDirs() []string
	exportSystemIncludeDirs() []string
	exportLdflags() []string
	exportLdlibs() []string
	exportLocalIncludeDirs() []string
	exportLocalSystemIncludeDirs() []string
	exportSharedLibs() []string
}
