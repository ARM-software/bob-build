/*
 * Copyright 2020-2023 Arm Limited.
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

/*
 * This file is included when Bob is being run as a standalone binary, i.e. for
 * the Ninja generator.
 */

package core

// Modules that produce content in the build output directory that may
// be referenced by other modules must implement the outputs() and
// implicitOutputs() functions. This structure supplies basic versions
// of these functions, where the modules just need to create the
// relevant lists.
//
// These must be set by the time GenerateBuildActions() completes.
type simpleOutputProducer struct {

	// List of all explicit outputs produced by this module, as we
	// expect to see them named in the generated build definition.
	// Whether these are relative or absolute paths will depend on the
	// generatorBackend in use. Where the generatorBackend requires
	// full paths, then BackendPaths (which use build system
	// variables) rather than explicit paths should be used.
	outs []string

	// List of all implicit outputs produced by this module, as we
	// expect to see them named in the generated build definition.
	// Whether these are relative or absolute paths will depend on the
	// generatorBackend in use. Where the generatorBackend requires
	// full paths, then BackendPaths (which use build system
	// variables) rather than explicit paths should be used.
	implicitOuts []string
}

func (m *simpleOutputProducer) implicitOutputs() []string {
	return m.implicitOuts
}

// Modules that produce headers in the build output directory that may
// be referenced by other modules must implement the genIncludeDirs()
// function. This structure supplies a basic version of this function,
// where the modules just need to create the relevant lists.
//
// This must be set by the time GenerateBuildActions() completes.
type headerProducer struct {
	// List of all include directories within the output directory
	// that are exported by this module. The directories are as we
	// expect to see them named in the generated build definition.
	// Whether these are relative or absolute paths will depend on the
	// generatorBackend in use. Where the generatorBackend requires
	// full paths, then BackendPaths (which use build system
	// variables) rather than explicit paths should be used.
	includeDirs []string
}

func (m *headerProducer) genIncludeDirs() []string {
	return m.includeDirs
}
