package toolchain

import (
	"github.com/ARM-software/bob-build/internal/utils"
)

type xcodeLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l xcodeLinker) GetTool() string {
	return l.tool
}

func (l xcodeLinker) GetFlags() []string {
	return l.flags
}

func (l xcodeLinker) GetLibs() []string {
	return l.libs
}

func (l xcodeLinker) KeepUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) DropUnusedDependencies() string {
	return ""
}

func (l xcodeLinker) SetRpathLink(path string) string {
	return ""
}

func (l xcodeLinker) SetVersionScript(path string) string {
	return ""
}

func (l xcodeLinker) SetRpath(path []string) string {
	return ""
}

func (l xcodeLinker) LinkWholeArchives(libs []string) string {
	return utils.Join(libs)
}

func (l xcodeLinker) KeepSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) DropSharedLibraryTransitivity() string {
	return ""
}

func (l xcodeLinker) GetForwardingLibFlags() string {
	return ""
}

func newXcodeLinker(tool string, flags, libs []string) (linker xcodeLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}
