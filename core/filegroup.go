/*
 * Copyright 2018-2023 Arm Limited.
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
	"strings"
	"sync"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type FileGroupProps struct {
	Srcs           []string
	Filegroup_srcs []string
}

type filegroup struct {
	moduleBase
	Properties struct {
		FileGroupProps
		Features
	}
}

func (m *filegroup) getSources(ctx blueprint.BaseModuleContext) (srcs []string) {
	srcs = m.Properties.Srcs
	return
}

func (m *filegroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getBackend(ctx).filegroupActions(m, ctx)
}

func (m *filegroup) shortName() string {
	return m.Name()
}

func (m *filegroup) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	for _, path := range m.Properties.Srcs {
		if strings.Contains(path, "..") {
			utils.Die("%s contains a relative uplink. Relative uplinks are not allowed in filegroup targets.", m.shortName())
		}
	}
	prefix := projectModuleDir(ctx)
	m.Properties.Srcs = utils.PrefixDirs(m.Properties.Srcs, prefix)
}

var (
	filegroupMap     = map[string][]string{}
	filegroupMapLock sync.RWMutex
)

func prepFilegroupMapMutator(mctx blueprint.BottomUpMutatorContext) {
	if l, ok := getLibrary(mctx.Module()); ok {
		filegroupMapLock.Lock()
		defer filegroupMapLock.Unlock()

		filegroupMap[mctx.ModuleName()] = l.Properties.Filegroup_srcs
	} else if fg, ok := mctx.Module().(*filegroup); ok {
		filegroupMapLock.Lock()
		defer filegroupMapLock.Unlock()

		filegroupMap[mctx.ModuleName()] = fg.Properties.Filegroup_srcs
	}
}

func expandFilegroup(d string, visited []string) []string {
	var filegroups []string
	if len(filegroupMap[d]) > 0 {
		for _, def := range filegroupMap[d] {
			if utils.Find(visited, def) >= 0 {
				utils.Die("filegroup module %s depends upon itself", def)
			}
			filegroups = append(filegroups, expandFilegroup(def, append(visited, def))...)
			filegroups = append(filegroups, def)
		}
	}
	return filegroups
}

func propogateFilegroupData(mctx blueprint.BottomUpMutatorContext) {
	if _, ok := getLibrary(mctx.Module()); ok {
		flattenedFilegroups := expandFilegroup(mctx.ModuleName(), []string{})
		var filegroups []string

		for i, el := range flattenedFilegroups {
			if utils.Find(flattenedFilegroups[i+1:], el) == -1 {
				filegroups = append(filegroups, el)
			}
		}

		mctx.AddDependency(mctx.Module(), filegroupTag, filegroups...)
	}
}

func filegroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &filegroup{}
	module.Properties.Features.Init(&config.Properties, FileGroupProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
