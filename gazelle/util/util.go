package util

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/language"
)

func IsChildFilepath(parent string, child string) (bool, error) {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false, err
	}
	if !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." {
		return true, nil
	}

	return false, nil
}

func MergeResults(args ...language.GenerateResult) (merged language.GenerateResult) {
	for _, r := range args {
		merged.Gen = append(merged.Gen, r.Gen...)
		merged.Empty = append(merged.Empty, r.Empty...)
		merged.Imports = append(merged.Imports, r.Imports...)
	}
	return
}
