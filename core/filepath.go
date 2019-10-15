/*
 * Copyright 2019 Arm Limited.
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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

// filePath encapsulates paths that may need to be used in different
// ways in source generation modules.
type filePath interface {
	// Full path to source used by ninja.
	// e.g. ${g.bob.SrcDir}/module1/subdir/file1.c
	//      ${g.bob.BuildDir}/gen/module2/subdir/file2.c
	//      ${g.bob.BuildDir}/bob.config
	buildPath() string
	// Short path used to calculate names of related outputs.
	// This may include the module directory.
	// e.g. module1/subdir/file1.c
	//      module2/subdir/file2.c
	//      bob.config
	localPath() string
	// Module directory associated with this file.
	// This path is valid from the build output directory.
	// e.g. ${g.bob.SrcDir}/module1
	//      ${g.bob.BuildDir}/gen/module2
	//      ${g.bob.BuildDir}
	moduleDir() string
}

// Represents a normal file in the source directory
type sourceFilePath struct {
	path      string
	module    string
	srcPrefix string
}

func (file sourceFilePath) buildPath() string {
	return filepath.Join(file.srcPrefix, file.path)
}

func (file sourceFilePath) localPath() string {
	return file.path
}

func (file sourceFilePath) moduleDir() string {
	return filepath.Join(file.srcPrefix, file.module)
}

func newSourceFilePath(path string, ctx blueprint.ModuleContext, g generatorBackend) filePath {
	return sourceFilePath{path, ctx.ModuleDir(), g.sourcePrefix()}
}

// Represents a file created in the generated output directory
type generatedFilePath struct {
	path    string
	lclPath string
	modDir  string
}

func (file generatedFilePath) buildPath() string {
	return file.path
}

func (file generatedFilePath) localPath() string {
	return file.lclPath
}

func (file generatedFilePath) moduleDir() string {
	return file.modDir
}

func newGeneratedFilePath(path string) filePath {
	// Identify the parts we need from the full path.
	//
	// Ideally we wouldn't need to do this - each module would return
	// generatedFilePaths.
	//
	// For generated paths, the backends set:
	// ${BuildDir}/gen/m.Name()/file.ext
	// $(TARGET_OUT_GEN)/STATIC_LIBRARIES/m.Name()/file.ext
	// $(HOST_OUT_GEN)/STATIC_LIBRARIES/m.Name()/file.ext
	//
	// Local path is anything from m.Name() (included)
	// Module dir is everything upto and including m.Name().
	pathElems := strings.Split(path, string(os.PathSeparator))
	if len(pathElems) < 4 {
		panic(fmt.Errorf("Path doesn't have as many elements as expected. %s", path))
	}
	lclPath := filepath.Join(pathElems[2:]...)
	modDir := filepath.Join(pathElems[:3]...)
	return generatedFilePath{path, lclPath, modDir}
}

// Handle special files (i.e. bob.config) in the generated output
// directory a bit differently.
func newSpecialFilePath(path string) filePath {
	// Identify the parts we need from the full path.
	//
	// Ideally we wouldn't need to do this - each module would return
	// filePaths.
	//
	// The special path should look like
	// ${BuildDir}/filename.ext
	// $(TARGET_OUT_GEN)/STATIC_LIBRARIES/directory/filename.ext
	//
	// Local path should just be the basename.
	// Module dir should be the directory.
	return generatedFilePath{path, filepath.Base(path), filepath.Dir(path)}
}
