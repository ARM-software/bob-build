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

package bpwriter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Implement types and helpers to record all the modules that we want
// to write into an Android.bp file.
//
// This is a really basic implementation that allows us to add key
// value pairs to a module, and store string, bools and string lists.
// Only a single level of nesting for properties is supported.

func indentString(depth int) string {
	return strings.Repeat(" ", depth*4)
}

var bpEscaper = strings.NewReplacer("\"", "\\\"")

func Escape(s string) string {
	return bpEscaper.Replace(s)
}

func EscapeList(slice []string) []string {
	slice = append([]string(nil), slice...)
	for i, s := range slice {
		slice[i] = Escape(s)
	}
	return slice
}

// Key-value pair for a property
type property struct {
	// Property key
	key string
	// Property value represented as a string
	value string
}

// Adjacent properties within an Android.bp file
type Group interface {
	AddString(name, value string)
	AddBool(name string, value bool)
	AddStringList(name string, list []string)
}

type group struct {
	// The name of this group. "" for the top level group.
	// This is necessary as we want to store group as an array
	// rather than a map, to maintain ordering.
	name string
	// Depth
	depth int
	// Properties in the group.
	// This is a list rather than a map to maintain ordering.
	props []property
}

var _ Group = (*group)(nil)

func (g *group) addProp(key, value string) {
	p := property{
		key:   key,
		value: value,
	}
	g.props = append(g.props, p)
}

// Add a string property to the property group
func (g *group) AddString(name, value string) {
	g.addProp(name, "\""+Escape(value)+"\"")
}

// Add a boolean property to the property group
func (g *group) AddBool(name string, value bool) {
	if value {
		g.addProp(name, "true")
	} else {
		g.addProp(name, "false")
	}
}

// Add a string list property to the property group
func (g *group) AddStringList(name string, list []string) {
	if len(list) == 0 {
		return
	}

	s := ""

	if len(list) > 1 {
		// Put each entry on a new line, indented
		s += "[\n"
		indent := indentString(g.depth + 1)
		for _, v := range list {
			s += indent + "\"" + Escape(v) + "\",\n"
		}
		// The list close is back-indented one tab
		s += indentString(g.depth) + "]"
	} else {
		// One entry. Put on a single line.
		s += "[\"" + Escape(list[0]) + "\"]"
	}
	g.addProp(name, s)
}

// Render the property group into a string
func (g *group) render(b *strings.Builder) {
	indent := indentString(g.depth)
	for _, p := range g.props {
		b.WriteString(indent + p.key + ": " + p.value + ",\n")
	}
}

// Arbitrary Android.bp module
//
// No locking for modules, as the creation of each module is done
// in a single thread.
type Module interface {
	Group
	NewGroup(name string) Group
}

type module struct {
	// Module type
	modType string

	// Module name
	name string

	// Top level properties of module
	group

	// Nested properties (1 level deep only)
	groups []group
}

var _ Module = (*module)(nil)

// Add a string property as a top level module property
func (m *module) AddString(name, value string) {
	m.group.AddString(name, value)
}

// Add a boolean property as a top level module property
func (m *module) AddBool(name string, value bool) {
	m.group.AddBool(name, value)
}

// Add a string list property as a top level module property
func (m *module) AddStringList(name string, list []string) {
	m.group.AddStringList(name, list)
}

// Create a property group in the module
func (m *module) NewGroup(name string) Group {
	g := group{}
	g.name = name
	g.depth = 2
	m.groups = append(m.groups, g)
	return &g
}

// Render the module into a string
func (m *module) render(b *strings.Builder) {
	indent := indentString(1)
	b.WriteString(m.modType + " {\n" + indent + "name: \"" + m.name + "\",\n")
	m.group.render(b)
	for _, group := range m.groups {
		b.WriteString(indent + group.name + ": {\n")
		group.render(b)
		b.WriteString(indent + "},\n")
	}
	b.WriteString("}\n\n")
}

func moduleFactory(modType, name string) *module {
	m := module{}
	m.modType = modType
	m.name = name
	m.group.name = ""
	m.group.depth = 1
	return &m
}

// Content for an Android.bp file
type File interface {
	NewModule(modType, name string) (Module, error)
	Render(b *strings.Builder)
}

type file struct {
	sync.Mutex
	modules map[string]*module
}

var _ File = (*file)(nil)

// Create a module
func (f *file) NewModule(modType, name string) (Module, error) {
	m := moduleFactory(modType, name)

	// Lock the addition to ensure parallel build actions can add
	// modules to the file.
	f.Lock()
	defer f.Unlock()

	if _, dup := f.modules[name]; !dup {
		f.modules[name] = m
	} else {
		err := fmt.Errorf("Duplicate module name (%s)", name)
		return nil, err
	}

	return m, nil
}

// Render all the modules in the file
func (f *file) Render(b *strings.Builder) {
	modNames := make([]string, len(f.modules))
	i := 0
	for name, _ := range f.modules {
		modNames[i] = name
		i = i + 1
	}
	sort.Strings(modNames)
	for _, name := range modNames {
		f.modules[name].render(b)
	}
}

func FileFactory() File {
	f := file{}
	f.modules = make(map[string]*module)
	return &f
}
