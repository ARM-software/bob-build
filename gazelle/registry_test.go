package plugin

import (
	"github.com/bazelbuild/bazel-gazelle/label"
	"testing"
)

func Test_register_module(t *testing.T) {
	registry := NewRegistry()
	testLabel := label.Label{Repo: "repo", Pkg: "some/pkg", Name: "m_name"}
	m := NewModule("m_name", "bob_binary", "some/pkg", "repo")

	registry.register(m)

	if !registry.nameExists("m_name") {
		t.Errorf("module %s not successfully registered", m.getName())
	}

	if !registry.labelExists(testLabel) {
		t.Errorf("module %s not successfully registered", m.getName())
	}

	if _, ok := registry.retrieveByPath("some/pkg"); !ok {
		t.Errorf("module %s not successfully registered", m.getName())
	}

}
