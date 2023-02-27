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
	"sync"

	"github.com/ARM-software/bob-build/internal/utils"
	"github.com/google/blueprint"
)

type filegroup struct {
	moduleBase
	Properties struct {
		SourceProps
		Features
	}
}

// Implementation check:
var _ sourceInterface = (*filegroup)(nil)

func (m *filegroup) getSourceFiles(ctx blueprint.BaseModuleContext) []string {
	return m.Properties.SourceProps.getSourceFiles(ctx)
}

func (m *filegroup) getSourceTargets(ctx blueprint.BaseModuleContext) []string {
	return m.Properties.SourceProps.getSourceTargets(ctx)
}

func (m *filegroup) getSourcesResolved(ctx blueprint.BaseModuleContext) []string {
	return m.Properties.SourceProps.getSourcesResolved(ctx)
}

func (m *filegroup) GenerateBuildActions(ctx blueprint.ModuleContext) {
	getBackend(ctx).filegroupActions(m, ctx)
}

func (m *filegroup) shortName() string {
	return m.Name()
}

func (m *filegroup) processPaths(ctx blueprint.BaseModuleContext, g generatorBackend) {
	m.Properties.SourceProps.processPaths(ctx, g)
}

var (
	filegroupMap     = map[string][]string{}
	filegroupMapLock sync.RWMutex
)

func prepFilegroupMapMutator(mctx blueprint.BottomUpMutatorContext) {
	moduleName := mctx.ModuleName()
	module := mctx.Module()
	if m, ok := module.(sourceInterface); ok {
		filegroupMapLock.Lock()
		defer filegroupMapLock.Unlock()
		filegroupMap[moduleName] = m.getSourceTargets(mctx)
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
		filegroups := utils.Unique(flattenedFilegroups)
		mctx.AddDependency(mctx.Module(), filegroupTag, filegroups...)
	}
}

func filegroupFactory(config *BobConfig) (blueprint.Module, []interface{}) {
	module := &filegroup{}
	module.Properties.Features.Init(&config.Properties, SourceProps{})
	return module, []interface{}{&module.Properties,
		&module.SimpleName.Properties}
}
