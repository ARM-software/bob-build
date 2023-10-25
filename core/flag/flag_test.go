package flag

import (
	"testing"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/stretchr/testify/assert"
)

func TestLinux(t *testing.T) {

	backend.Setup(config.GetEnvironmentVariables(),
		config.CreateMockConfig(map[string]interface{}{
			"builder_ninja": true,
		}),
	)

	raw_local_path := "local/foo"
	raw_global_path := "/global/foo"

	t.Run("SimpleFlag", func(t *testing.T) {
		f := FromString("-Wall", TypeTransitive)
		assert.Equal(t, "-Wall", f.ToString())

		assert.True(t, f.IsType(TypeTransitive)) // matches exactly

		assert.True(t, f.MatchesType(TypeTransitive|TypeExported)) //loosely matches

		assert.True(t, f.IsNotType(TypeExported))
		assert.True(t, f.IsNotType(TypeExported|TypeTransitive))

		assert.Equal(t, f.Raw(), "-Wall")
		assert.Equal(t, f.Raw(), f.ToString()) //Simple case raw == string
	})

	t.Run("FromIncludePath", func(t *testing.T) {
		tag := TypeIncludeLocal
		f := FromIncludePath(raw_local_path, tag)
		assert.Equal(t, "-I${SrcDir}/local/foo", f.ToString())
		assert.Equal(t, f.Type(), tag|TypeInclude)
		assert.Equal(t, f.Raw(), raw_local_path)

		tag |= TypeIncludeSystem
		f = FromIncludePath(raw_local_path, tag)
		assert.Equal(t, "-isystem ${SrcDir}/local/foo", f.ToString())

		tag ^= TypeIncludeLocal
		f = FromIncludePath(raw_global_path, tag)
		assert.Equal(t, "-isystem /global/foo", f.ToString())

		tag ^= TypeIncludeSystem
		f = FromIncludePath(raw_global_path, tag)
		assert.Equal(t, "-I/global/foo", f.ToString())
	})
}
