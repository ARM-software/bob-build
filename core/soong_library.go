// +build soong

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

	"android/soong/android"
	"android/soong/cc"

	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/utils"
)

type ccLibraryCommonProps struct {
	Name               *string
	Stem               *string
	Srcs               []string
	Exclude_srcs       []string
	Generated_sources  []string
	Generated_headers  []string
	Cflags             []string
	Include_dirs       []string
	Local_include_dirs []string
	Static_libs        []string
	Whole_static_libs  []string
	Shared_libs        []string
	Ldflags            []string
}

type ccStaticOrSharedProps struct {
	Export_include_dirs []string
}

// Convert between Bob module names, and the name we will give the generated
// cc_library module. This is required when a module supports being built on
// host and target; we cannot create two modules with the same name, so
// instead, we use the `shortName()` (which may include a `__host` or
// `__target` suffix) to disambiguate, and use the `stem` property to fix up
// the output filename.
func ccModuleName(mctx android.TopDownMutatorContext, name string) string {
	m, _ := mctx.GetDirectDep(name + bobModuleSuffix)

	if l, ok := getLibrary(m); ok {
		return l.shortName()
	}

	return m.Name()
}

func ccModuleNames(mctx android.TopDownMutatorContext, nameLists ...[]string) []string {
	ccModules := []string{}
	for _, nameList := range nameLists {
		for _, name := range nameList {
			ccModules = append(ccModules, ccModuleName(mctx, name))
		}
	}
	return ccModules
}

func (l *library) getExportedCflags(mctx android.TopDownMutatorContext) []string {
	visited := map[string]bool{}
	cflags := []string{}
	mctx.VisitDirectDeps(func(dep android.Module) {
		if !(mctx.OtherModuleDependencyTag(dep) == wholeStaticDepTag ||
			mctx.OtherModuleDependencyTag(dep) == staticDepTag ||
			mctx.OtherModuleDependencyTag(dep) == sharedDepTag ||
			mctx.OtherModuleDependencyTag(dep) == reexportLibsTag) {
			return
		} else if _, ok := visited[dep.Name()]; ok {
			// VisitDirectDeps will visit a module once for each
			// dependency. We've already done this module.
			return
		}

		if sl, ok := getLibrary(dep); ok {
			cflags = append(cflags, sl.Properties.Export_cflags...)
		}
	})
	return cflags
}

func (l *library) getGeneratedSources(mctx android.TopDownMutatorContext) (srcs []string) {
	mctx.VisitDirectDepsWithTag(generatedSourceTag, func(dep android.Module) {
		switch dep.(type) {
		case *generateSource:
		case *transformSource:
		default:
			panic(fmt.Errorf("Dependency %s of %s is not a generated source",
				dep.Name(), l.Name()))
		}

		srcs = append(srcs, dep.Name())
	})
	return
}

func (l *library) getGeneratedHeaders(mctx android.TopDownMutatorContext) (headers []string) {
	mctx.VisitDirectDepsWithTag(generatedHeaderTag, func(dep android.Module) {
		switch dep.(type) {
		case *generateSource:
		case *transformSource:
		default:
			panic(fmt.Errorf("Dependency %s of %s is not a generated source",
				dep.Name(), l.Name()))
		}

		headers = append(headers, dep.Name())
	})
	return
}

func (l *library) setupCcLibraryProps(mctx android.TopDownMutatorContext) *ccLibraryCommonProps {
	if len(l.Properties.Export_include_dirs) > 0 {
		panic(fmt.Errorf("Module %s exports non-local include dirs %v - this is not supported",
			mctx.ModuleName(), l.Properties.Export_include_dirs))
	}

	cflags := utils.NewStringSlice(l.Properties.Cflags,
		l.Properties.Export_cflags, l.getExportedCflags(mctx))

	props := &ccLibraryCommonProps{
		Name:               proptools.StringPtr(l.shortName()),
		Stem:               proptools.StringPtr(l.Name()),
		Srcs:               utils.Filter(utils.IsCompilableSource, l.Properties.Srcs),
		Generated_sources:  l.getGeneratedSources(mctx),
		Generated_headers:  l.getGeneratedHeaders(mctx),
		Exclude_srcs:       l.Properties.Exclude_srcs,
		Cflags:             cflags,
		Include_dirs:       l.Properties.Include_dirs,
		Local_include_dirs: l.Properties.Local_include_dirs,
		Static_libs:        ccModuleNames(mctx, l.Properties.ResolvedStaticLibs),
		Whole_static_libs:  ccModuleNames(mctx, l.Properties.Whole_static_libs),
		Shared_libs:        ccModuleNames(mctx, l.Properties.Shared_libs, l.Properties.Export_shared_libs),
		Ldflags:            l.Properties.Ldflags,
	}

	return props
}

// Create a module which only builds on the device. The closest thing Soong
// provides will also allow building on the host, which is not quite what we
// want.
func libraryTargetStaticFactory() android.Module {
	module, library := cc.NewLibrary(android.DeviceSupported)
	library.BuildOnlyStatic()
	return module.Init()
}

func (l *staticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	if !isEnabled(l) {
		return
	}

	commonProps := l.setupCcLibraryProps(mctx)

	libProps := &ccStaticOrSharedProps{
		// Soong's `export_include_dirs` field is relative to the module dir.
		Export_include_dirs: l.Properties.Export_local_include_dirs,
	}

	switch l.Properties.TargetType {
	case tgtTypeHost:
		mctx.CreateModule(android.ModuleFactoryAdaptor(cc.LibraryHostStaticFactory), commonProps, libProps)
	case tgtTypeTarget:
		mctx.CreateModule(android.ModuleFactoryAdaptor(libraryTargetStaticFactory), commonProps, libProps)
	}
}

// Create a module which only builds on the device. The closest thing Soong
// provides will also allow building on the host, which is not quite what we
// want.
func libraryTargetSharedFactory() android.Module {
	module, library := cc.NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	return module.Init()
}

func (l *sharedLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	if !isEnabled(l) {
		return
	}

	commonProps := l.setupCcLibraryProps(mctx)

	libProps := &ccStaticOrSharedProps{
		// Soong's `export_include_dirs` field is relative to the module dir.
		Export_include_dirs: l.Properties.Export_local_include_dirs,
	}
	stripProps := &cc.StripProperties{}
	if l.strip() {
		stripProps.Strip.All = proptools.BoolPtr(true)
	}

	switch l.Properties.TargetType {
	case tgtTypeHost:
		mctx.CreateModule(android.ModuleFactoryAdaptor(cc.LibraryHostSharedFactory),
			commonProps, libProps, stripProps)
	case tgtTypeTarget:
		mctx.CreateModule(android.ModuleFactoryAdaptor(libraryTargetSharedFactory),
			commonProps, libProps, stripProps)
	}
}

// From Soong's cc/binary.go. This is needed here because it is not exported by Soong.
func binaryHostFactory() android.Module {
	module, _ := cc.NewBinary(android.HostSupported)
	return module.Init()
}

// Like libraryTargetStaticFactory, create a module which is only buildable on the device.
func binaryTargetFactory() android.Module {
	module, _ := cc.NewBinary(android.DeviceSupported)
	return module.Init()
}

func (b *binary) soongBuildActions(mctx android.TopDownMutatorContext) {
	if !isEnabled(b) {
		return
	}

	commonProps := b.setupCcLibraryProps(mctx)
	stripProps := &cc.StripProperties{}
	if b.strip() {
		stripProps.Strip.All = proptools.BoolPtr(true)
	}

	switch b.Properties.TargetType {
	case tgtTypeHost:
		mctx.CreateModule(android.ModuleFactoryAdaptor(binaryHostFactory),
			commonProps, stripProps)
	case tgtTypeTarget:
		mctx.CreateModule(android.ModuleFactoryAdaptor(binaryTargetFactory),
			commonProps, stripProps)
	}

}
