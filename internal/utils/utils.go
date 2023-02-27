/*
 * Copyright 2018-2021, 2023 Arm Limited.
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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	headerRegexp        = regexp.MustCompile(`\.(h|hpp|inc)$`)
	compileSourceRegexp = regexp.MustCompile(`\.(c|s|cpp|cc|S)$`)
)

// Does the input string look like it is a header file?
func IsHeader(s string) bool {
	return headerRegexp.MatchString(s)
}

// Does the input string look like it is a source file?
func IsNotHeader(s string) bool {
	return !headerRegexp.MatchString(s)
}

// IsCompilableSource checks if filename extension is a compiled one
func IsCompilableSource(s string) bool {
	return compileSourceRegexp.MatchString(s)
}

// IsNotCompilableSource checks if filename extension isn't a compiled one
func IsNotCompilableSource(s string) bool {
	return !compileSourceRegexp.MatchString(s)
}

// Prefixes a string to every item in a list
func PrefixAll(list []string, prefix string) []string {
	output := []string{}
	for _, s := range list {
		output = append(output, prefix+s)
	}
	return output
}

// Removes prefix from every item in the list
func StripPrefixAll(list []string, prefix string) []string {
	output := []string{}
	for _, s := range list {
		output = append(output, strings.TrimPrefix(s, prefix))
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

func SortedKeysByteSlice(m map[string][]byte) []string {
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

func Unique(list []string) (ret []string) {
	return AppendUnique([]string{}, list)
}

func ListsContain(x string, lists ...[]string) bool {
	for _, list := range lists {
		if Contains(list, x) {
			return true
		}
	}
	return false
}

func Filter(predicate func(string) bool, lists ...[]string) (ret []string) {
	for _, list := range lists {
		for _, s := range list {
			if predicate(s) {
				ret = append(ret, s)
			}
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

	output := make([]string, len(destination))
	copy(output, destination)

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

// `cmd` is the command that will be executed
// Return true if it contains a reference to the argument expansion `arg`
func ContainsArg(cmd string, k string) bool {
	// argument ref can be done as ${arg} or $arg
	if strings.Contains(cmd, "${"+k+"}") || strings.Contains(cmd, "$"+k) {
		return true
	}
	return false
}

// cmd is the command that will be executed
// args contains potential argument that may occur in ${}
// This function will remove unused arguments from the map
func StripUnusedArgs(args map[string]string, cmd string) {
	for k := range args {
		if !ContainsArg(cmd, k) {
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

// Join multiple lists of strings. This replaces appending multiple arrays
// together before calling strings.Join().
func Join(lists ...[]string) string {
	var sb strings.Builder
	const sep = " "
	first := true

	for _, list := range lists {
		listJoined := strings.Join(list, sep)
		if len(listJoined) > 0 {
			if !first {
				sb.WriteString(sep)
			} else {
				first = false
			}
			sb.WriteString(listJoined)
		}
	}

	return sb.String()
}

// IsExecutable returns true if the given file exists and is executable
func IsExecutable(fname string) bool {
	if fi, err := os.Stat(fname); !os.IsNotExist(err) && (fi.Mode()&0111 != 0) {
		return true
	}
	return false
}

// NewStringSlice initialises a new slice from the input lists, which are concatenated.
// The in-built append function modifies the slice buffer of the existing slice.
// This means that using append has side-effect on the first list.
// The purpose of this function is to avoid those side-effects.
func NewStringSlice(lists ...[]string) []string {
	// Checkout utils test for example why this is different to append()
	sumSize := 0
	for _, list := range lists {
		sumSize += len(list)
	}
	allLists := make([]string, 0, sumSize)
	for _, list := range lists {
		allLists = append(allLists, list...)
	}
	return allLists
}

// Exit the program, printing a message to stderr.
// Deferred functions will not execute.
func Exit(exitCode int, err string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, err+"\n", a...)
	os.Exit(exitCode)
}

func Die(err string, a ...interface{}) {
	Exit(1, err, a...)
}

// FlattenPath produces a filename containing no slashes from a path.
func FlattenPath(s string) string {
	return strings.Replace(s, "/", "__", -1)
}

// Expand function is similar to os.Expand, with one difference: curly braces
// around variable names are compulsory - this only replaces recognized
// variable references. In particular, '$' followed by a character other than
// '{' will not be affected by expansion. E.g. "$a" will remain "$a" after
// expansion.
func Expand(s string, mapping func(string) string) (res string) {
	const (
		outsideRef = iota
		enteringRef
		insideRef
	)
	state := outsideRef
	variable := ""
	for _, c := range s {
		switch state {
		case outsideRef:
			if c == '$' {
				state = enteringRef
			} else {
				res += string(c)
			}
		case enteringRef:
			if c == '{' {
				state = insideRef
			} else {
				res += "$" + string(c)
				state = outsideRef
			}
		case insideRef:
			if c == '}' {
				res += mapping(variable)
				variable = ""
				state = outsideRef
			} else {
				variable += string(c)
			}
		}
	}
	if state == enteringRef {
		res += "$"
	} else if state == insideRef {
		res += "${" + variable
	}
	return
}

func SplitPath(path string) (components []string) {
	pathSep := string(os.PathSeparator)

	if path == "" {
		return []string{}
	} else if path == pathSep {
		return []string{pathSep}
	}

	// Ignore trailing slashes
	if path[len(path)-1:] == pathSep {
		path = path[:len(path)-1]
	}

	dir, file := filepath.Split(path)

	if dir == "" {
		return []string{file}
	} else if dir == pathSep {
		// Treat subdirs of the root directory as single components, e.g. /var -> ["/var"].
		return []string{path}
	}

	return append(SplitPath(dir), file)
}
