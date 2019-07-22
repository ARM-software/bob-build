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
}

func (l *library) setupCcLibraryProps(mctx android.TopDownMutatorContext) (bool, *ccLibraryCommonProps) {
	if !isEnabled(l) {
		return false, nil
	}

	// For now, only build target libraries
	if l.Properties.TargetType != tgtTypeTarget {
		return false, nil
	}

	props := &ccLibraryCommonProps{
		Name:               proptools.StringPtr(l.Name()),
		Srcs:               utils.Filter(utils.IsCompilableSource, l.Properties.Srcs),
		Exclude_srcs:       l.Properties.Exclude_srcs,
		Cflags:             l.Properties.Cflags,
		Include_dirs:       l.Properties.Include_dirs,
		Local_include_dirs: l.Properties.Local_include_dirs,
		Static_libs:        utils.NewStringSlice(l.Properties.Static_libs, l.Properties.Export_static_libs),
		Whole_static_libs:  l.Properties.Whole_static_libs,
	}

	return true, props
}

func (l *staticLibrary) soongBuildActions(mctx android.TopDownMutatorContext) {
	enabled, props := l.setupCcLibraryProps(mctx)
	if !enabled {
		return
	}

	mctx.CreateModule(android.ModuleFactoryAdaptor(cc.LibraryStaticFactory), props)
}

func (b *binary) soongBuildActions(mctx android.TopDownMutatorContext) {
	enabled, props := b.setupCcLibraryProps(mctx)
	if !enabled {
		return
	}

	mctx.CreateModule(android.ModuleFactoryAdaptor(cc.BinaryFactory), props)
}
