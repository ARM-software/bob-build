package registry

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
)

type MockRegistrable struct {
	Name         string
	RelativePath string
	Label        label.Label
}

var _ Registrable = (*MockRegistrable)(nil)

func (m *MockRegistrable) GetName() string         { return m.Name }
func (m *MockRegistrable) GetRelativePath() string { return m.RelativePath }
func (m *MockRegistrable) GetLabel() label.Label   { return m.Label }

func Test_register_module(t *testing.T) {
	registry := NewRegistry()
	testLabel := label.Label{Repo: "", Pkg: "some/pkg", Name: "m_name"}
	m := &MockRegistrable{
		Name:         "m_name",
		RelativePath: "some/pkg",
		Label:        testLabel,
	}

	registry.Register(m)

	if !registry.NameExists("m_name") {
		t.Errorf("module %s not successfully registered", m.GetName())
	}

	if !registry.LabelExists(testLabel) {
		t.Errorf("module %s not successfully registered", m.GetName())
	}

	if _, ok := registry.RetrieveByPath("some/pkg"); !ok {
		t.Errorf("module %s not successfully registered", m.GetName())
	}
}
