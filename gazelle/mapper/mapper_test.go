package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BasicMapperTests(t *testing.T) {

	m := NewMapper()

	assert.Nil(t, m.FromValue(":foo"))
	assert.Nil(t, m.FromValue("foo"))

	fooLabel := MakeLabel(":foo", "")
	bazLabel := MakeLabel(":baz", "some/other/path")
	m.Map(fooLabel, ":foo")
	assert.Equal(t, m.FromValue("foo"), fooLabel)
	assert.Equal(t, m.FromValue(":foo"), fooLabel)

	m.Map(fooLabel, "foo") //This is a no op
	assert.Equal(t, m.FromValue("foo"), fooLabel)
	assert.Equal(t, m.FromValue(":foo"), fooLabel)

	m.Map(bazLabel, "foo") //remap works
	assert.Equal(t, m.FromValue("foo"), bazLabel)
	assert.Equal(t, m.FromValue(":foo"), bazLabel)
}
