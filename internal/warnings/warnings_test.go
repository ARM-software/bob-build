/*
 * Copyright 2022 Arm Limited.
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

package warnings

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureStderr(f func()) string {
	r, w, err := os.Pipe()

	if err != nil {
		panic(err)
	}

	stderr := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	f()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()

}

func TestWarningDefault(t *testing.T) {
	const expected string = `BpFile,BpModule,WarningAction,WarningMessage,WarningCategory
A/build.bp,gen_table,ignore,"Warning ocurred, will warn!",UserWarning
`
	var msg strings.Builder

	wr := New(&msg, "")
	wr.Warn(UserWarning, "A/build.bp", "gen_table", "Warning ocurred, will warn!")

	assert.Equal(t, expected, msg.String())
}

func TestWarning(t *testing.T) {
	const expected string = `BpFile,BpModule,WarningAction,WarningMessage,WarningCategory
A/build.bp,gen_table,warning,Warning ocurred!,UserWarning
B/build.bp,gen_binary,ignore,Warning ocurred!,RelativeUpLinkWarning
`
	const expectedStderr string = "A/build.bp:gen_table: warning: Warning ocurred! [UserWarning]\n"
	var msg strings.Builder

	wr := New(&msg, "UserWarning:W")

	stderrOutput := captureStderr(func() {
		wr.Warn(UserWarning, "A/build.bp", "gen_table", "Warning ocurred!")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary", "Warning ocurred!")
	})

	//assert.Equal(t, expected, msg.String())
	assert.Equal(t, expectedStderr, stderrOutput)
}

func TestErrorWarning(t *testing.T) {
	const expected string = `BpFile,BpModule,WarningAction,WarningMessage,WarningCategory
A/build.bp,gen_table,error,Wrong target!,UserWarning
B/build.bp,gen_binary,warning,Warning ocurred!,RelativeUpLinkWarning
`
	const expectedStderr string = `A/build.bp:gen_table: error: Wrong target! [UserWarning]
B/build.bp:gen_binary: warning: Warning ocurred! [RelativeUpLinkWarning]
`
	var msg strings.Builder

	wr := New(&msg, "UserWarning:E RelativeUpLinkWarning:W")
	stderrOutput := captureStderr(func() {
		wr.Warn(UserWarning, "A/build.bp", "gen_table", "Wrong target!")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary", "Warning ocurred!")
	})

	assert.Equal(t, expected, msg.String())
	assert.Equal(t, expectedStderr, stderrOutput)
}

func TestFilterOverwriteCategory(t *testing.T) {
	const expected string = "Overriding warning category not allowed: 'UserWarning:W'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "UserWarning:E RelativeUpLinkWarning:W UserWarning:W")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestFilterOverwriteWildcard(t *testing.T) {
	const expected string = "Overriding wildcard (*) not allowed: '*:W'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "*:E RelativeUpLinkWarning:W *:W")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestErrorAllWarnings(t *testing.T) {
	const expectedStderr string = `A/build.bp:gen_table: error: Wrong target! [UserWarning]
B/build.bp:gen_binary: warning: Warning ocurred! [RelativeUpLinkWarning]
`
	const expected string = `BpFile,BpModule,WarningAction,WarningMessage,WarningCategory
A/build.bp,gen_table,error,Wrong target!,UserWarning
B/build.bp,gen_binary,warning,Warning ocurred!,RelativeUpLinkWarning
`
	var msg strings.Builder

	wr := New(&msg, "*:E RelativeUpLinkWarning:W")
	stderrOutput := captureStderr(func() {
		wr.Warn(UserWarning, "A/build.bp", "gen_table", "Wrong target!")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary", "Warning ocurred!")
	})

	assert.Equal(t, expected, msg.String())
	assert.Equal(t, expectedStderr, stderrOutput)
}

func TestWrongFilterCategory(t *testing.T) {
	const expected string = "Wrong filter category 'Ridiculous:E'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "Ridiculous:E")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestWrongFilterAction(t *testing.T) {
	const expected string = "Wrong filter action 'DirectPathsWarning:D'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "DirectPathsWarning:D")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestWrongFilter(t *testing.T) {
	const expected string = "Wrong warnings filter expression 'DirectPathsWarning'\n"
	var msg strings.Builder

	stderrOutput := captureStderr(func() {
		New(&msg, "DirectPathsWarning RelativeUpLinkWarning:W")
	})

	assert.Equal(t, expected, stderrOutput)
}

func TestCheckIfErrors(t *testing.T) {
	var msg strings.Builder

	wr := New(&msg, "UserWarning:E RelativeUpLinkWarning:W")
	captureStderr(func() {
		wr.Warn(UserWarning, "A/build.bp", "gen_table", "Wrong target!")
		wr.Warn(RelativeUpLinkWarning, "B/build.bp", "gen_binary", "Warning ocurred!")
		assert.Equal(t, 1, wr.ErrorWarnings())
		wr.Warn(UserWarning, "ABC/build.bp", "gen_lib", "Another wrong target!")
		wr.Warn(RelativeUpLinkWarning, "BCD/build.bp", "gen_binary_two", "Warning ocurred!")
		assert.Equal(t, 2, wr.ErrorWarnings())
	})
}
