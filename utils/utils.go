/*
 * Copyright 2018 Arm Limited.
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

package utils

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	headerRegexp = regexp.MustCompile(`\.(h|hpp|inc)$`)
)

// Does the input string look like it is a header file?
func IsHeader(s string) bool {
	return headerRegexp.MatchString(s)
}

// Does the input string look like it is a source file?
func IsSource(s string) bool {
	return !headerRegexp.MatchString(s)
}

// Prefixes a string to every item in a list
func PrefixAll(list []string, prefix string) []string {
	output := []string{}
	for _, s := range list {
		output = append(output, prefix+s)
	}
	return output
}

// Prefixes a directory to every file in a list
// The returned file paths are the shortest paths (../ removed)
func PrefixDirs(paths []string, dir string) []string {
	output := []string{}
	for _, p := range paths {
		output = append(output, filepath.Join(dir, p))
	}
	return output
}

func SortedKeys(m map[string]string) []string {
	keys := make([]string, len(m))

	i := 0
	for key := range m {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	return keys
}

func SortedKeysBoolMap(m map[string]bool) []string {
	keys := make([]string, len(m))

	i := 0
	for key := range m {
		keys[i] = key
		i++
	}

	sort.Strings(keys)

	return keys
}

/* Identifies whether the array 'list' contains the string 'x'. */
func Contains(list []string, x string) bool {
	for _, y := range list {
		if y == x {
			return true
		}
	}
	return false
}

func Filter(ss []string, predicate func(string) bool) (ret []string) {
	for _, s := range ss {
		if predicate(s) {
			ret = append(ret, s)
		}
	}
	return
}

// return s after removing elements found in t
func Difference(s []string, t []string) []string {
	var diff []string
	for _, element := range s {
		if !Contains(t, element) {
			diff = append(diff, element)
		}
	}
	return diff
}

func AppendIfUnique(destination []string, value string) []string {
	if !Contains(destination, value) {
		return append(destination, value)
	}

	return destination
}

func AppendUnique(destination []string, source []string) []string {
	if len(source) < 1 {
		return destination
	}

	output := destination

	for _, value := range source {
		output = AppendIfUnique(output, value)
	}

	return output
}

// Find returns the smallest index i at which x == a[i],
// or -1 if there is no such index.
func Find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

func Remove(a []string, x string) []string {
	position := Find(a, x)
	if position != -1 {
		return append(a[:position], a[position+1:]...)
	}
	return a
}

// Return a reversed a string array.
// This ought to be able to be done generically for any array type by using interfaces,
// but we only need to handle strings for now
func Reversed(in []string) []string {
	size := len(in)
	last := size - 1
	out := make([]string, size)
	for i, s := range in {
		out[last-i] = s
	}
	return out
}

// cmd is the command that will be executed
// args contains potential argument that may occur in ${}
// This function will remove unused arguments from the map
func StripUnusedArgs(args map[string]string, cmd string) {
	for k := range args {
		// argument ref can be done as ${arg} or $arg
		if !strings.Contains(cmd, "${"+k+"}") && !strings.Contains(cmd, "$"+k) {
			delete(args, k)
		}
	}
}

func Trim(args []string) []string {
	out := []string{}
	for _, element := range args {
		if trim := strings.TrimSpace(element); len(trim) > 0 {
			out = append(out, trim)
		}
	}
	return out
}
