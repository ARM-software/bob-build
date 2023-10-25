package file

import (
	"testing"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/stretchr/testify/assert"
)

func TestLinuxGroups(t *testing.T) {

	backend.Setup(config.GetEnvironmentVariables(),
		config.CreateMockConfig(map[string]interface{}{
			"builder_ninja": true,
		}),
	)

	paths := Paths{
		NewPath("file0", "0", TypeUnset),
		NewPath("file1", "1", TypeGenerated),
		NewPath("file2", "2", TypeTool),
		NewPath("file3", "3", TypeBinary),
		NewPath("file4", "4", TypeExecutable),
		NewPath("file5", "5", TypeImplicit),
		NewPath("file.c", "6", TypeUnset),
		NewPath("file.cpp", "7", TypeUnset),
		NewPath("file.s", "8", TypeUnset),
		NewPath("file.h", "9", TypeUnset),
		NewPath("file.a", "10", TypeUnset),
		NewPath("file.so", "11", TypeUnset),
		NewPath("file.c", "12", TypeCompilable),
	}

	t.Run("ToStringSliceBuildPaths", func(t *testing.T) {
		expected := []string{
			"${SrcDir}/file0",
			"${BuildDir}/gen/1/file1",
			"${SrcDir}/file2",
			"${BuildDir}/3/executable/file3",
			"${BuildDir}/4/executable/file4",
			"${SrcDir}/file5",
			"${SrcDir}/file.c",
			"${SrcDir}/file.cpp",
			"${SrcDir}/file.s",
			"${SrcDir}/file.h",
			"${BuildDir}/10/static/file.a",
			"${BuildDir}/11/shared/file.so",
			"${SrcDir}/file.c",
		}

		assert.Equal(t, expected, paths.ToStringSlice(
			func(p Path) string {
				return p.BuildPath()
			}))
	})

	t.Run("ToStringSliceBuildPathsFiltered", func(t *testing.T) {
		expected := []string{
			"${BuildDir}/10/static/file.a",
		}

		assert.Equal(t, expected, paths.ToStringSliceIf(
			func(p Path) bool {
				return p.IsType(TypeArchive)
			},
			func(p Path) string {
				return p.BuildPath()
			}))
	})

}
