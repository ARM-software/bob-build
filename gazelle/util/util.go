package util

import (
	"os"
	"path/filepath"
	"strings"
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
