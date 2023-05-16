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

package core

import (
	"path"
	"path/filepath"
)

type FileType uint32

const (
	FileTypeUnset              = 0
	FileTypeGenerated FileType = 1 << iota
	FileTypeTool
	FileTypeBinary
	FileTypeExecutable
	FileTypeImplicit
	FileTypeC
	FileTypeCpp
	FileTypeAsm
	FileTypeHeader

	// Masks:
	FileTypeCompilable = FileTypeC | FileTypeCpp | FileTypeAsm
)

type FilePath struct {
	backendPath string // either absolute location of the source tree, or generated file build root for AOSP/Linux respectively

	namespacePath string
	relativePath  string
	tag           FileType // tag to indicate type

}

func (file FilePath) RelBuildPath() string {
	if file.IsType(FileTypeGenerated) {
		// We want to preserve /gen/ in the path when using relative build path
		return filepath.Join("gen", file.namespacePath, file.relativePath)
	} else {
		return filepath.Join(file.namespacePath, file.relativePath)
	}
}

func (file FilePath) BuildPath() string {
	return filepath.Join(file.backendPath, file.namespacePath, file.relativePath)
}

func (file FilePath) UnScopedPath() string {
	return file.relativePath
}

func (file FilePath) ScopedPath() string {
	return filepath.Join(file.namespacePath, file.relativePath)
}

func (file FilePath) Scope() string {
	return file.namespacePath
}

func (file FilePath) Ext() string {
	return path.Ext(file.relativePath)
}

func (file FilePath) IsType(ft FileType) bool {
	return (file.tag & ft) != 0
}

func (file FilePath) IsNotType(ft FileType) bool {
	return ((file.tag & ft) ^ ft) != 0
}

var FileNoNameSpace string = ""

func newFile(relativePath string, namespace string, g generatorBackend, tag FileType) FilePath {
	// TODO: remove generator backend here
	// TODO: add noncompiled dep tag
	switch path.Ext(relativePath) {
	case ".s", ".S":
		tag |= FileTypeAsm
	case ".c":
		tag |= FileTypeC
	case ".cc", ".cpp", ".cxx":
		tag |= FileTypeCpp
	case ".h", ".hpp":
		tag |= FileTypeHeader
		// TODO: .so .a .o
	}

	var backendPath string
	var scopedPath string
	if (tag & (FileTypeBinary | FileTypeExecutable)) != 0 {
		backendPath = g.buildDir()
	} else if (tag & FileTypeGenerated) != 0 {
		backendPath = filepath.Join(g.buildDir(), "gen")
		scopedPath = namespace
	} else {
		backendPath = g.sourceDir()
		scopedPath = FileNoNameSpace
	}

	return FilePath{
		backendPath:   backendPath,
		namespacePath: scopedPath,
		relativePath:  relativePath,
		tag:           tag,
	}
}
