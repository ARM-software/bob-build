/*
 * Copyright 2018-2020 Arm Limited.
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

	"github.com/google/blueprint"
)

type defaults struct {
	library
}

func (m *defaults) supportedVariants() []tgtType {
	return []tgtType{tgtTypeHost, tgtTypeTarget}
}

func (m *defaults) disable() {
	panic("disable() called on Default")
}

func (m *defaults) setVariant(variant tgtType) {
	m.library.setVariant(variant)
}

func (m *defaults) getSplittableProps() *SplittableProps {
	return m.library.getSplittableProps()
}

func (m *defaults) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

func defaultsFactory(config *bobConfig) (blueprint.Module, []interface{}) {
	module := &defaults{}
	return module.LibraryFactory(config, module)
}

var defaultDepTag = dependencyTag{name: "default"}

// Modules implementing defaultable can refer to bob_defaults via the
// `defaults` property
type defaultable interface {
	build() *Build
	features() *Features
	defaults() []string
}

func defaultDepsMutator(mctx blueprint.BottomUpMutatorContext) {
	if l, ok := mctx.Module().(defaultable); ok {
		mctx.AddDependency(mctx.Module(), defaultDepTag, l.defaults()...)
	}
	if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		if len(gsc.Properties.Flag_defaults) > 0 {
			tgt := gsc.Properties.Target
			if !(tgt == tgtTypeHost || tgt == tgtTypeTarget) {
				panic(fmt.Errorf("Module %s uses flag_defaults '%v' but has invalid target type '%s'",
					mctx.ModuleName(), gsc.Properties.Flag_defaults, tgt))
			}
			mctx.AddDependency(mctx.Module(), defaultDepTag, gsc.Properties.Flag_defaults...)
		}
	}
}
