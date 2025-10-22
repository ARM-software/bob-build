package toolchain

import (
	"fmt"
	"strings"

	"github.com/ARM-software/bob-build/internal/utils"
)

type customLinker struct {
	tool  string
	flags []string
	libs  []string
}

func (l customLinker) GetTool() string {
	return l.tool
}

func (l customLinker) GetFlags() []string {
	return l.flags
}

func (l customLinker) GetLibs() []string {
	return l.libs
}

func (l customLinker) KeepUnusedDependencies() string {
	return "-Wl,--no-as-needed"
}

func (l customLinker) DropUnusedDependencies() string {
	return "-Wl,--as-needed"
}

func (l customLinker) KeepSharedLibraryTransitivity() string {
	return ""
}

func (l customLinker) DropSharedLibraryTransitivity() string {
	return ""
}

func (l customLinker) GetForwardingLibFlags() string {
	return ""
}

func (l customLinker) SetRpathLink(path string) string {
	return "-Wl,-rpath-link," + path
}

func (l customLinker) SetVersionScript(path string) string {
	return "-Wl,--version-script," + path
}

func (l customLinker) SetRpath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("-Wl,--enable-new-dtags")
	for _, p := range paths {
		fmt.Fprintf(&b, ",-rpath=%s", p)
	}
	return b.String()
}

func (l customLinker) LinkWholeArchives(libs []string) string {
	if len(libs) == 0 {
		return ""
	}
	return fmt.Sprintf("-Wl,--whole-archive %s -Wl,--no-whole-archive", utils.Join(libs))
}

func newCustomLinker(tool string, flags, libs []string) (linker customLinker) {
	linker.tool = tool
	linker.flags = flags
	linker.libs = libs
	return
}
