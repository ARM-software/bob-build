/*
 * Copyright 2019-2023 Arm Limited.
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

package file

import (
	"path"
	"path/filepath"
)

type Type uint32

const (
	TypeUnset          = 0
	TypeGenerated Type = 1 << iota
	TypeTool
	TypeBinary
	TypeExecutable
	TypeImplicit
	TypeC
	TypeCpp
	TypeAsm
	TypeHeader

	// Masks:
	TypeCompilable = TypeC | TypeCpp | TypeAsm
)

type Path struct {
	backendPath string // either absolute location of the source tree, or generated file build root for AOSP/Linux respectively

	namespacePath string
	relativePath  string
	tag           Type // tag to indicate type

}

func (file Path) RelBuildPath() string {
	if file.IsType(TypeGenerated) {
		// We want to preserve /gen/ in the path when using relative build path
		return filepath.Join("gen", file.namespacePath, file.relativePath)
	} else {
		return filepath.Join(file.namespacePath, file.relativePath)
	}
}

func (file Path) BuildPath() string {
	return filepath.Join(file.backendPath, file.namespacePath, file.relativePath)
}

func (file Path) UnScopedPath() string {
	return file.relativePath
}

func (file Path) ScopedPath() string {
	return filepath.Join(file.namespacePath, file.relativePath)
}

func (file Path) Scope() string {
	return file.namespacePath
}

func (file Path) Ext() string {
	return path.Ext(file.relativePath)
}

func (file Path) IsType(ft Type) bool {
	return (file.tag & ft) != 0
}

func (file Path) IsNotType(ft Type) bool {
	return ((file.tag & ft) ^ ft) != 0
}

var FileNoNameSpace string = ""

func NewPath(relativePath string, namespace string, tag Type) Path {
	return GetFactory().New(relativePath, namespace, tag)
}
