/*
 * Copyright 2020, 2022 Arm Limited.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
