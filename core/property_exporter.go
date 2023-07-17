package core

import "github.com/ARM-software/bob-build/core/flag"

type propertyExporter interface {
	flag.Provider // Eventually the below functions will be removed
	exportLdflags() []string
	exportLdlibs() []string
	exportSharedLibs() []string
}
