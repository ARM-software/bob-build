package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyAdd(t *testing.T) {
	tc := New()
	tc.Add("foo/bar/baz", "tc1")
	tc.Add("foo/bar", "tc2")
}

func Test_UnregisteredDefault(t *testing.T) {
	tc := New()
	assert.Equal(t, "", tc.Get("."))
}

func TestCommon(t *testing.T) {
	tc := New()
	tc.Add("foo/bar/baz", "baz")
	tc.Add("foo/bar", "bar1")
	tc.Add("foo/bar", "bar2")
	tc.Add(".", "root")

	t.Run("DirectHit", func(t *testing.T) {
		assert.Equal(t, "root", tc.Get("."))
	})

	t.Run("RootDefault", func(t *testing.T) {
		assert.Equal(t, "root", tc.Get("does/not/exist"))
	})

	t.Run("OrderOfRegister", func(t *testing.T) {
		assert.Equal(t, "bar1", tc.Get("foo/bar"))
		assert.Equal(t, "bar1", tc.Get("foo/bar/child"))
	})

}
