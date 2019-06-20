/*
 * Copyright 2018-2019 Arm Limited.
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

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Create a ConfigProperties type that we can use for testing. This
// avoids having to setup JSON input files for each test.
func setupTestConfig(m map[string]string) *configProperties {
	properties := &configProperties{}

	// We can ignore ConfigPropertiesJson, as that is just used to read in the JSON file.
	properties.stringMap = m

	return properties
}

type testProperties struct {
	// Normal strings
	StrA string
	StrB string
	StrC string

	// Arrays
	StrArray []string

	// Strings referencing a boolean value
	B1 string
	B2 string

	// Pointer to Strings
	RefA *string
	RefB *string
}

// Check that templates are expanded in a simple property structure
func TestApplyTemplate(t *testing.T) {
	config := setupTestConfig(map[string]string{
		"a": "alpha",
		"b": "beta",
		"c": "gamma",
		"t": "1",
		"f": "0",
	})

	arr := []string{
		"{{.a}}", "{{.b}}", "{{.c}}",
	}
	refA := "{{.a}}{{.t}}"
	refB := "{{.b}}{{.f}}"

	props := testProperties{
		"{{.a}}",
		"{{.b}}",
		"{{.c}}",
		arr,
		"{{.t}}",
		"{{.f}}",
		&refA,
		&refB,
	}
	ApplyTemplate(&props, config)

	// Check templates are expanded in normal strings
	assert.Equalf(t, props.StrA, "alpha", "StrA incorrect")
	assert.Equalf(t, props.StrB, "beta", "StrB incorrect")
	assert.Equalf(t, props.StrC, "gamma", "StrC incorrect")

	// Check 'booleans'. These are actually strings as far as the
	// template code is concerned
	assert.Equalf(t, props.B1, "1", "B1 incorrect")
	assert.Equalf(t, props.B2, "0", "B2 incorrect")

	// Check templates have been expanded in arrays of strings
	assert.Equalf(t, arr[0], "alpha", "arr[0] incorrect")
	assert.Equalf(t, arr[1], "beta", "arr[1] incorrect")
	assert.Equalf(t, arr[2], "gamma", "arr[2] incorrect")

	// Check templates have been expanded in pointers to strings
	assert.Equalf(t, refA, "alpha1", "refA incorrect")
	assert.Equalf(t, refB, "beta0", "refB incorrect")
}

type testNestedProperties struct {
	A, B testProperties
}

// Check that templates are expanded in a nested property structure
func TestApplyTemplateNested(t *testing.T) {
	config := setupTestConfig(map[string]string{
		"a": "alpha",
		"b": "beta",
		"c": "gamma",
		"t": "1",
		"f": "0",
	})

	arr := []string{
		"{{.a}}", "{{.b}}", "{{.c}}",
	}
	arrB := []string{
		"{{.b}}", "{{.c}}", "{{.a}}",
	}
	refA := "{{.a}}{{.t}}"
	refB := "{{.b}}{{.f}}"

	props := testNestedProperties{
		testProperties{
			"{{.a}}",
			"{{.b}}",
			"{{.c}}",
			arr,
			"{{.t}}",
			"{{.f}}",
			&refA,
			&refB,
		},
		testProperties{
			"{{.c}}",
			"{{.a}}",
			"{{.b}}",
			arrB,
			"{{.f}}",
			"{{.t}}",
			nil, // Pointers to string can be nil
			nil,
		},
	}

	ApplyTemplate(&props, config)

	// Check templates are expanded in normal strings
	assert.Equalf(t, props.A.StrA, "alpha", "A.StrA incorrect")
	assert.Equalf(t, props.A.StrB, "beta", "A.StrB incorrect")
	assert.Equalf(t, props.A.StrC, "gamma", "A.StrC incorrect")
	assert.Equalf(t, props.B.StrA, "gamma", "B.StrA incorrect")
	assert.Equalf(t, props.B.StrB, "alpha", "B.StrB incorrect")
	assert.Equalf(t, props.B.StrC, "beta", "B.StrC incorrect")

	// Check 'booleans'. These are actually strings as far as the
	// template code is concerned
	assert.Equalf(t, props.A.B1, "1", "A.B1 incorrect")
	assert.Equalf(t, props.A.B2, "0", "A.B2 incorrect")
	assert.Equalf(t, props.B.B1, "0", "B.B1 incorrect")
	assert.Equalf(t, props.B.B2, "1", "B.B2 incorrect")

	// Check templates have been expanded in arrays of strings
	assert.Equalf(t, arr[0], "alpha", "arr[0] incorrect")
	assert.Equalf(t, arr[1], "beta", "arr[1] incorrect")
	assert.Equalf(t, arr[2], "gamma", "arr[2] incorrect")
	assert.Equalf(t, arrB[0], "beta", "arrB[0] incorrect")
	assert.Equalf(t, arrB[1], "gamma", "arrB[1] incorrect")
	assert.Equalf(t, arrB[2], "alpha", "arrB[2] incorrect")

	// Check templates have been expanded in pointers to strings
	assert.Equalf(t, refA, "alpha1", "refA incorrect")
	assert.Equalf(t, refB, "beta0", "refB incorrect")
}
