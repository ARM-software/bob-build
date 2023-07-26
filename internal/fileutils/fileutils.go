package fileutils

// Useful routines for file reading/writing

import (
	"io/ioutil"
	"os"
	"strings"
)

func WriteIfChanged(filename string, sb *strings.Builder) error {
	mustWrite := true
	text := sb.String()

	// If any errors occur trying to determine the state of the existing file,
	// just write the new file
	fileinfo, err := os.Stat(filename)
	if err == nil {
		if fileinfo.Size() == int64(sb.Len()) {
			current, err := ioutil.ReadFile(filename)
			if err == nil {
				if string(current) == text {
					// No need to write
					mustWrite = false
				}
			}
		}
	}

	if mustWrite {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		file.WriteString(text)
		file.Close()
	}

	return nil
}
