package flag

import (
	"testing"

	"github.com/ARM-software/bob-build/core/backend"
	"github.com/ARM-software/bob-build/core/config"
	"github.com/stretchr/testify/assert"
)

func TestCollections(t *testing.T) {

	backend.Setup(config.GetEnvironmentVariables(),
		config.CreateMockConfig(map[string]interface{}{
			"builder_ninja": true,
		}),
		nil, // logger is nil here, not used in these tests
	)

	flags := Flags{
		FromString("Unset", TypeUnset),
		FromString("Include|     |             |        ", TypeInclude),
		FromString("Include|     |             |Exported", TypeInclude|TypeExported),
		FromString("Include|     |IncludeSystem|        ", TypeInclude|TypeIncludeSystem),
		FromString("Include|     |IncludeSystem|Exported", TypeInclude|TypeIncludeSystem|TypeExported),
		FromString("Include|Local|             |        ", TypeInclude|TypeIncludeLocal),
		FromString("Include|Local|             |Exported", TypeInclude|TypeIncludeLocal|TypeExported),
		FromString("Include|Local|IncludeSystem|        ", TypeInclude|TypeIncludeLocal|TypeIncludeSystem),
		FromString("Include|Local|IncludeSystem|Exported", TypeInclude|TypeIncludeLocal|TypeIncludeSystem|TypeExported),
		FromString("Asm", TypeAsm),
		FromString("C", TypeC),
		FromString("Cpp", TypeCpp),
		FromString("CC", TypeCC),
		FromString("CC|Exported", TypeCC|TypeExported),
		FromString("Linker", TypeLinker),
		FromString("Linker|Exported", TypeLinker|TypeExported),
		FromString("Compilable|Exported", TypeCompilable|TypeExported),
		FromString("Compilable", TypeCompilable),
		FromString("Transitive", TypeTransitive),
		FromString("C|Transitive", TypeC|TypeTransitive),
	}

	t.Run("filters", func(t *testing.T) {
		no_exports := flags.Filtered(
			func(f Flag) bool {
				return !f.MatchesType(TypeExported)
			},
		)

		no_exports.ForEach(func(f Flag) {
			assert.False(t, f.IsType(TypeExported))
		})
	})

	t.Run("tag_groups", func(t *testing.T) {
		grouped := flags.GroupByType(TypeTransitive)
		assert.False(t, grouped[0].MatchesType(TypeTransitive))
		assert.True(t, grouped[len(grouped)-1].MatchesType(TypeTransitive))
	})

}
