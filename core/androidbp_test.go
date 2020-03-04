/*
 * Copyright 2020 Arm Limited.
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

	"github.com/ARM-software/bob-build/internal/bpwriter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockModule struct {
	mock.Mock
	bpwriter.Module
}

func (m *MockModule) AddString(name, value string) {
	m.Called(name, value)
}

func (m *MockModule) AddStringList(name string, values []string) {
	m.Called(name, values)
}

func Test_addCFlags(t *testing.T) {
	m := new(MockModule)

	m.On("AddString", "c_std", "c11")
	m.On("AddString", "cpp_std", "c++11")
	m.On("AddString", "instruction_set", "arm")
	m.On("AddStringList", "cflags", []string{"-cl-no-signed-zeros"})
	m.On("AddStringList", "conlyflags", []string(nil))
	m.On("AddStringList", "cppflags", []string(nil))

	cflags := []string{"-marm", "-mx32", "-cl-no-signed-zeros"}
	conlyFlags := []string{"-std=c11"}
	cxxFlags := []string{"-std=c++11"}

	addCFlags(m, cflags, conlyFlags, cxxFlags)

	m.AssertExpectations(t)
}

func Test_addCFlags2(t *testing.T) {
	m := new(MockModule)

	m.On("AddString", "c_std", "c17")
	m.On("AddString", "cpp_std", "c++17")
	m.On("AddString", "instruction_set", "thumb")
	m.On("AddStringList", "cflags", []string(nil))
	m.On("AddStringList", "conlyflags", []string(nil))
	m.On("AddStringList", "cppflags", []string(nil))

	cflags := []string{"-mthumb", "-mx32"}
	conlyFlags := []string{"-std=c17"}
	cxxFlags := []string{"-std=c++17"}

	addCFlags(m, cflags, conlyFlags, cxxFlags)

	m.AssertExpectations(t)
}

func Test_addCFlags3(t *testing.T) {
	m := new(MockModule)

	m.On("AddString", "cpp_std", "c++17")
	m.On("AddStringList", "cflags", []string(nil))
	m.On("AddStringList", "conlyflags", []string(nil))
	m.On("AddStringList", "cppflags", []string(nil))

	cflags := []string{"-mx32"}
	conlyFlags := []string{}
	cxxFlags := []string{"-std=c++17"}

	addCFlags(m, cflags, conlyFlags, cxxFlags)

	m.AssertExpectations(t)
}

func Test_addCFlags4(t *testing.T) {
	m := new(MockModule)

	cflags := []string{"-marm", "-mthumb"}
	conlyFlags := []string{}
	cxxFlags := []string{}

	err := addCFlags(m, cflags, conlyFlags, cxxFlags)

	assert.Equal(t, err.Error(), "Both thumb and no thumb (arm) options are specified")
}
