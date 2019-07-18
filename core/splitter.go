/*
 * Copyright 2018-2019 Arm Limited.
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
	"errors"
	"fmt"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/abstr"
)

// SplittableProps are embedded by modules which can be split into multiple variants
type SplittableProps struct {
	Host_supported   *bool
	Target_supported *bool
}

// If a Module implements this interface, then it will be split into
// the different variants by the splitterMutator
type splittable interface {
	// Retrieve all the different variations to create
	supportedVariants() []tgtType

	// Disables the module is no variations supported
	disable()

	// Set the particular variant
	setVariant(tgtType)

	// Get the properties related to which variants are available
	getSplittableProps() *SplittableProps
}

// Traverse the core properties of defaults to find out which variations are
// supported.
func supportedVariantsMutator(mctx abstr.TopDownMutatorContext) {
	sp, ok := abstr.Module(mctx).(splittable)
	if !ok {
		return
	}

	// It's not valid to specify host_supported or target_supported in a
	// target: {} or host: {} section
	if moduleWithBuildProps, ok := abstr.Module(mctx).(moduleWithBuildProps); ok {
		b := moduleWithBuildProps.build()
		if b.Host.SplittableProps.Host_supported != nil ||
			b.Host.SplittableProps.Target_supported != nil ||
			b.Target.SplittableProps.Host_supported != nil ||
			b.Target.SplittableProps.Target_supported != nil {
			panic(fmt.Errorf("target-specific [host|target]_supported in module %s",
				mctx.ModuleName()))
		}
	}

	// Defaults are always split into both variants
	if _, isDefaults := abstr.Module(mctx).(*defaults); isDefaults {
		return
	}

	visited := map[string]bool{}

	abstr.WalkDeps(mctx, func(dep blueprint.Module, parent blueprint.Module) bool {
		if mctx.OtherModuleDependencyTag(dep) == defaultDepTag {
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

			err := proptools.PrependMatchingProperties([]interface{}{sp.getSplittableProps()},
				&def.Properties.SplittableProps, nil)
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					panic(err)
				}
			}
			return true
		}
		return false
	})
}

func tgtToString(tgts []tgtType) []string {
	variants := make([]string, len(tgts))
	for i, v := range tgts {
		variants[i] = string(v)
	}
	return variants
}

// Creates all the supported variants of splittable modules, including defaults.
func splitterMutator(mctx blueprint.BottomUpMutatorContext) {
	if s, ok := mctx.Module().(splittable); ok {
		variants := tgtToString(s.supportedVariants())
		if len(variants) == 0 {
			s.disable()
		} else {
			modules := mctx.CreateVariations(variants...)
			for i, v := range variants {
				newsplit, ok := modules[i].(splittable)
				if !ok {
					panic(errors.New("newly created variation is not splittable - should not happen"))
				}
				newsplit.setVariant(tgtType(v))
			}
		}
	}
}
