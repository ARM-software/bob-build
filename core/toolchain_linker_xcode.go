package core

import (
	"github.com/ARM-software/bob-build/internal/utils"
)

type xcodeLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l xcodeLinker) getTool() string {
	return l.tool
}

func (l xcodeLinker) getFlags() []string {
	return l.flags
}

func (l xcodeLinker) getLibs() []string {
	return l.libs
}

func (l xcodeLinker) keepUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) dropUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) setRpathLink(path string) string {
	return ""
}

func (l xcodeLinker) setVersionScript(path string) string {
	return ""
}

func (l xcodeLinker) setRpath(path []string) string {
	return ""
}

func (l xcodeLinker) linkWholeArchives(libs []string) string {
	return utils.Join(libs)
}

func (l xcodeLinker) keepSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) dropSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) getForwardingLibFlags() string {
	return ""
}

func newXcodeLinker(tool string, flags, libs []string) (linker xcodeLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}
