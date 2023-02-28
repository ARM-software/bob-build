package plugin

import "github.com/bazelbuild/bazel-gazelle/label"

type BobModule struct {
	bobName, relativePath string
	bazelLabel            label.Label
}

func (m BobModule) getName() string {
	return m.bobName
}

func (m BobModule) getRelativePath() string {
	return m.relativePath
}

func (m BobModule) getLabel() label.Label {
	return m.bazelLabel
}
