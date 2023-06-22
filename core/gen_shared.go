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

package core

import (
	"github.com/ARM-software/bob-build/core/file"
	"github.com/ARM-software/bob-build/core/flag"
	"github.com/google/blueprint"
)

type generateSharedLibrary struct {
	generateLibrary
	fileNameExtension string
}

// Verify that the following interfaces are implemented
var _ FileProvider = (*generateSharedLibrary)(nil)
var _ generateLibraryInterface = (*generateSharedLibrary)(nil)
var _ singleOutputModule = (*generateSharedLibrary)(nil)
var _ sharedLibProducer = (*generateSharedLibrary)(nil)
var _ splittable = (*generateSharedLibrary)(nil)
var _ blueprint.Module = (*generateSharedLibrary)(nil)

func (m *generateSharedLibrary) generateInouts(ctx blueprint.ModuleContext, g generatorBackend) []inout {
	return generateLibraryInouts(m, ctx, g, m.Properties.Headers)
}

func (m *generateSharedLibrary) implicitOutputs() []string {
	return m.OutFiles().ToStringSliceIf(
		// TODO: ideally we should just check for `TypeImplicit` here,
		// but currently set up to mirror existing behaviour
		func(f file.Path) bool { return f.IsNotType(file.TypeShared) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *generateSharedLibrary) outputs() []string {
	return m.OutFiles().ToStringSliceIf(
		// TODO: fixme, this outputs headers as well so we need to filter it somewhere
		func(f file.Path) bool { return f.IsNotType(file.TypeImplicit) && f.IsType(file.TypeShared) },
		func(f file.Path) string { return f.BuildPath() })
}

func (m *generateSharedLibrary) filesToInstall(ctx blueprint.BaseModuleContext) []string {
	return m.outputs()
}

func (m *generateSharedLibrary) OutFiles() (files file.Paths) {
	gc, _ := getGenerateCommon(m)
	files = append(files, gc.OutFiles()...)

	files = append(files, file.NewPath(m.outputFileName(), m.Name(), file.TypeGenerated|file.TypeInstallable))

	for _, h := range m.Properties.Headers {
		fp := file.NewPath(h, m.Name(), file.TypeGenerated|file.TypeHeader)
		files = append(files, fp)
	}

	return
}

func (m *generateSharedLibrary) FlagsOut() (flags flag.Flags) {
	gc, _ := getGenerateCommon(m)
	for _, str := range gc.Properties.Export_gen_include_dirs {
		flags = append(flags, flag.FromGeneratedIncludePath(str, m, flag.TypeExported))
	}
	return
}

//// Support generateLibraryInterface

func (m *generateSharedLibrary) libExtension() string {
	return m.fileNameExtension
}

//// Support blueprint.Module

func (m *generateSharedLibrary) GenerateBuildActions(ctx blueprint.ModuleContext) {
	if isEnabled(m) {
		getGenerator(ctx).genSharedActions(m, ctx)
	}
}

//// Support singleOutputModule

func (m *generateSharedLibrary) outputFileName() string {
	return m.altName() + m.libExtension()
}

//// Support sharedLibProducer

func (m *generateSharedLibrary) getTocName() string {
	return m.outputFileName() + tocExt
}

func (m generateSharedLibrary) GetProperties() interface{} {
	return m.generateLibrary.Properties
}

//// Factory functions

func genSharedLibFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &generateSharedLibrary{}
	module.ModuleGenerateCommon.init(&config.Properties, GenerateProps{},
		GenerateLibraryProps{})

	if config.Properties.GetBool("osx") {
		module.fileNameExtension = ".dylib"
	} else {
		module.fileNameExtension = ".so"
	}
	return module, []interface{}{
		&module.SimpleName.Properties,
		&module.ModuleGenerateCommon.Properties,
		&module.Properties,
	}
}
