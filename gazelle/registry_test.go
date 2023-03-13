package plugin

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"testing"
)

func Test_register_module(t *testing.T) {
	registry := NewRegistry()
	testLabel := label.Label{Repo: "repo", Pkg: "some/pkg", Name: "m_name"}
	m := NewBobModule("m_name", "bob_binary", "some/pkg", "repo")
	registry.register(m)
	if !registry.nameExists("m_name") {
		t.Errorf("module %d not successfully registered", m.getName())
	}
	if !registry.labelExists(testLabel) {
		t.Errorf("module %d not successfully registered", m.getName())
	}
	if registry.retrieveByPath("some/pkg") == nil {
		t.Errorf("module %d not successfully registered", m.getName())
	}
}
