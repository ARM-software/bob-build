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
	Srcs               []string
	Exclude_srcs       []string
	Cflags             []string
	Include_dirs       []string
	Local_include_dirs []string
	Static_libs        []string
	Whole_static_libs  []string
	Ldflags            []string
}

type ccStaticOrSharedProps struct {
	Export_include_dirs []string
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

func (l *library) setupCcLibraryProps(mctx android.TopDownMutatorContext) (bool, *ccLibraryCommonProps) {
	if !isEnabled(l) {
		return false, nil
	}

	if len(l.Properties.Export_include_dirs) > 0 {
		panic(fmt.Errorf("Module %s exports non-local include dirs %v - this is not supported",
			mctx.ModuleName(), l.Properties.Export_include_dirs))
	}

	// For now, only build target libraries
	if l.Properties.TargetType != tgtTypeTarget {
		return false, nil
	}

	cflags := utils.NewStringSlice(l.Properties.Cflags,
		l.Properties.Export_cflags, l.getExportedCflags(mctx))

	props := &ccLibraryCommonProps{
		Name:               proptools.StringPtr(l.Name()),
		Srcs:               utils.Filter(utils.IsCompilableSource, l.Properties.Srcs),
		Exclude_srcs:       l.Properties.Exclude_srcs,
		Cflags:             cflags,
		Include_dirs:       l.Properties.Include_dirs,
		Local_include_dirs: l.Properties.Local_include_dirs,
		Static_libs:        l.Properties.ResolvedStaticLibs,
		Whole_static_libs:  l.Properties.Whole_static_libs,
		Ldflags:            l.Properties.Ldflags,
	}

	return true, props
}

func (l *staticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	enabled, commonProps := l.setupCcLibraryProps(mctx)
	if !enabled {
		return
	}

	libProps := &ccStaticOrSharedProps{
		// Soong's `export_include_dirs` field is relative to the module dir.
		Export_include_dirs: l.Properties.Export_local_include_dirs,
	}

	mctx.CreateModule(android.ModuleFactoryAdaptor(cc.LibraryStaticFactory), commonProps, libProps)
}

func (b *binary) soongBuildActions(mctx android.TopDownMutatorContext) {
	enabled, commonProps := b.setupCcLibraryProps(mctx)
	if !enabled {
		return
	}

	mctx.CreateModule(android.ModuleFactoryAdaptor(cc.BinaryFactory), commonProps)
}
