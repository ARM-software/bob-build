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
	"github.com/google/blueprint/proptools"
)

// Applies default options
func defaultApplierMutator(mctx blueprint.TopDownMutatorContext) {
	// This method walks down the dependency list to include all defaults that include other defaults
	// with the ones further down the tree being applied first.
	// Walkdeps is a preorder depth-first search - meaning a parent is visited before children, and children
	// is visited before siblings.
	_, isDefaults := mctx.Module().(*defaults)
	if isDefaults {
		return
	}

	var build *Build

	if target, ok := mctx.Module().(defaultable); ok {
		build = target.build()
	} else if gsc, ok := getGenerateCommon(mctx.Module()); ok {
		build = &gsc.Properties.FlagArgsBuild
	} else {
		// Not defaultable.
		return
	}

	visited := map[string]bool{}

	mctx.WalkDeps(func(dep blueprint.Module, parent blueprint.Module) bool {
		if mctx.OtherModuleDependencyTag(dep) == defaultDepTag {
			//print("Visiting " + mctx.OtherModuleName(dep) + " for dependency " + mctx.ModuleName() + "\n")
			def, ok := dep.(*defaults)
			if !ok {
				panic(fmt.Errorf("module %s in %s's defaults is not a default",
					dep.Name(), mctx.ModuleName()))
			}

			// Only visit each default once
			if _, ok := visited[dep.Name()]; ok {
				return false
			}
			visited[dep.Name()] = true

			// Defaults are more generic, so we prepend to the
			// core module properties.
			//
			// Note: when prepending (pointers to) bools we copy
			// the value if the dst is nil, otherwise the dst
			// value is left alone.
			err := proptools.PrependMatchingProperties([]interface{}{&build.BuildProps}, &def.build().BuildProps, nil)
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					panic(err)
				}
			}

			return true // This return value indicates if we want to continue visiting children.
		}
		return false
	})
}

// Modules implementing featurable support the use of features and templates.
type featurable interface {
	topLevelProperties() []interface{}
	features() *Features
}

func templateApplierMutator(mctx blueprint.TopDownMutatorContext) {
	module := mctx.Module()
	cfg := getConfig(mctx)

	if m, ok := module.(featurable); ok {
		cfgProps := &cfg.Properties

		// TemplateApplier mutator is run before TargetApplier, so we
		// need to apply templates with the core set, as well as
		// host-specific and target-specific sets (where applicable).
		props := append([]interface{}{}, m.topLevelProperties()...)

		if ts, ok := module.(targetSpecificLibrary); ok {
			host := ts.getTargetSpecific(tgtTypeHost)
			target := ts.getTargetSpecific(tgtTypeTarget)

			props = append(props, host.getTargetSpecificProps())
			props = append(props, target.getTargetSpecificProps())
		}

		for _, p := range props {
			ApplyTemplate(p, cfgProps)
		}
	}
}

// Used to map a set of properties to destination properties
type propmap struct {
	dst []interface{}
	src *Features
}

// Applies feature specific properties within each module
func featureApplierMutator(mctx blueprint.TopDownMutatorContext) {
	module := mctx.Module()
	cfg := getConfig(mctx)

	if m, ok := module.(featurable); ok {
		cfgProps := &cfg.Properties

		// FeatureApplier mutator is run first. We need to flatten the
		// feature specific properties in the core set, and where
		// supported, the host-specific and target-specific set.
		var props = []propmap{propmap{m.topLevelProperties(), m.features()}}

		if ts, ok := module.(targetSpecificLibrary); ok {
			host := ts.getTargetSpecific(tgtTypeHost)
			target := ts.getTargetSpecific(tgtTypeTarget)

			var tgtprops = []propmap{
				propmap{[]interface{}{host.getTargetSpecificProps()}, &host.Features},
				propmap{[]interface{}{target.getTargetSpecificProps()}, &target.Features},
			}
			props = append(props, tgtprops...)

		}

		for _, prop := range props {
			// Feature specific properties get added after core properties.
			//
			// Note: when appending (pointers to) bools we always override
			// the dst value. i.e. feature-specific value takes precedence.
			err := prop.src.AppendProps(prop.dst, cfgProps)
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					panic(err)
				}
			}
		}
	}
}
