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
	"testing"
	"unicode"
)

func assertTrue(t *testing.T, cond bool, msg string) {
	if !cond {
		t.Errorf("%s: Condition is not true", msg)
	}
}

func assertFalse(t *testing.T, cond bool, msg string) {
	if cond {
		t.Errorf("%s: Condition is not true", msg)
	}
}

func Test_IsHeader(t *testing.T) {
	assertTrue(t, IsHeader("bla.hpp"), "bla.hpp")
	assertFalse(t, IsHeader("bla.bin"), "bla.bin")
}

func Test_IsSource(t *testing.T) {
	assertTrue(t, IsSource("bla.c"), "bla.hpp")
	assertFalse(t, IsSource("bla.h"), "bla.bin")
}

func assertArraysEqual(t *testing.T, test []string, correct []string) {
	if len(test) != len(correct) {
		t.Errorf("Length mismatch: %d != %d", len(test), len(correct))
	}

	for i := range test {
		if test[i] != correct[i] {
			t.Errorf("Bad prefix for index %d: '%s' != '%s'", i, test[i], correct[i])
		}
	}
}

func Test_PrefixAll(t *testing.T) {
	if len(PrefixAll([]string{}, "myprefix")) != 0 {
		t.Errorf("Incorrect handling of empty list")
	}

	in := []string{"abc def", ";1234	;''"}
	prefix := "!>@@\""
	correct := []string{"!>@@\"abc def", "!>@@\";1234	;''"}
	assertArraysEqual(t, PrefixAll(in, prefix), correct)
}

func Test_PrefixDirs(t *testing.T) {
	if len(PrefixDirs([]string{}, "myprefix")) != 0 {
		t.Errorf("Incorrect handling of empty list")
	}

	in := []string{"src/foo.c", "include/bar.h"}
	prefix := "$(LOCAL_PATH)"
	correct := []string{"$(LOCAL_PATH)/src/foo.c", "$(LOCAL_PATH)/include/bar.h"}
	assertArraysEqual(t, PrefixDirs(in, prefix), correct)
}

func Test_SortedKeys(t *testing.T) {
	in := map[string]string{
		"Zebra":    "grass",
		"aardvark": "insects",
		"./a.out":  "bits",
	}
	assertArraysEqual(t, SortedKeys(in), []string{"./a.out", "Zebra", "aardvark"})
}

func Test_SortedKeysBoolMap(t *testing.T) {
	in := map[string]bool{
		"Alphabetic characters should appear after numbers": true,
		"2 + 2 = 5": false,
	}
	assertArraysEqual(t, SortedKeysBoolMap(in),
		[]string{"2 + 2 = 5",
			"Alphabetic characters should appear after numbers"})
}

func Test_Contains(t *testing.T) {
	assertFalse(t, Contains([]string{"a", "b", "c"}, "yellow"), "alphabet")
	assertTrue(t, Contains([]string{"a", "b", "c"}, "c"), "alphabet")
	assertFalse(t, Contains([]string{}, "anything"), "empty list")
	assertFalse(t, Contains([]string{}, ""), "empty strings")
	assertTrue(t, Contains([]string{""}, ""), "empty strings")
}

func Test_Filter(t *testing.T) {
	in := []string{"Alpha", "beta", "Gamma", "Delta", "epsilon"}
	filtered := Filter(in, func(elem string) bool {
		return unicode.IsUpper(rune(elem[0]))
	})
	assertArraysEqual(t, filtered, []string{"Alpha", "Gamma", "Delta"})
}

func Test_Difference(t *testing.T) {
	in := []string{"1", "1", "2", "3", "5", "8", "13", "21"}
	sub := []string{"2", "8", "21"}
	correct := []string{"1", "1", "3", "5", "13"}
	assertArraysEqual(t, Difference(in, sub), correct)
}

func Test_AppendUnique(t *testing.T) {
	// AppendIfUnique is tested via AppendUnique.
	assertArraysEqual(t,
		AppendUnique([]string{},
			[]string{"test"}),
		[]string{"test"})
	assertArraysEqual(t,
		AppendUnique([]string{"ab", "cd"},
			[]string{"", "", "ef"}),
		[]string{"ab", "cd", "", "ef"})
	assertArraysEqual(t,
		AppendUnique([]string{"ab", "cd"},
			[]string{"cd", "ab"}),
		[]string{"ab", "cd"})
}

func Test_Find(t *testing.T) {
	assertTrue(t, Find([]string{"abc", "abcde"}, "abcd") == -1, "Incorrect index")
	assertTrue(t, Find([]string{"abc", "abcde"}, "abcde") == 1, "Incorrect index")
}

func Test_Remove(t *testing.T) {
	assertArraysEqual(t, Remove([]string{"abc", "abcde"}, "abcd"),
		[]string{"abc", "abcde"})
	assertArraysEqual(t, Remove([]string{"abc", "abcde"}, "abcde"),
		[]string{"abc"})
	assertArraysEqual(t, Remove([]string{"abc", "abcde"}, "abc"),
		[]string{"abcde"})
}

func Test_Reversed(t *testing.T) {
	assertArraysEqual(t, Reversed([]string{}),
		[]string{})
	assertArraysEqual(t, Reversed([]string{""}),
		[]string{""})
	assertArraysEqual(t, Reversed([]string{"123", "234"}),
		[]string{"234", "123"})
	assertArraysEqual(t, Reversed([]string{"", "234", "..<>"}),
		[]string{"..<>", "234", ""})
}

func Test_StripUnusedArgs(t *testing.T) {
	args := map[string]string{
		"compiler": "gcc",
		"args":     "-Wall -Werror -c",
		"depfile":  "deps.d",
		"wrapper":  "ccache",
		"in":       "source.c",
		"out":      "source.o",
	}
	StripUnusedArgs(args, "${compiler} -o ${out} ${in} ${args}")
	assertArraysEqual(t, SortedKeys(args), []string{"args", "compiler", "in", "out"})
}

func Test_Trim(t *testing.T) {
	assertArraysEqual(t, Trim([]string{"", " hello ", "world", "	"}),
		[]string{"hello", "world"})
}

func Test_Join(t *testing.T) {
	assertTrue(t,
		Join() == "",
		"Empty join should yield an empty string")
	assertTrue(t,
		Join([]string{"Hello", "world"}) == "Hello world",
		"Didn't concatenate two words")

	// Here, there's a 3rd space before "spaces", because strings.Join()
	// adds one for the empty string, but utils.Join() doesn't.
	assertTrue(t,
		Join([]string{"this is", " surrounded by "},
			[]string{"", "spaces"}, []string{}) ==
			"this is  surrounded by   spaces",
		"Surrounding space not handled")
}
