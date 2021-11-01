/*
 * Copyright 2018-2021 Arm Limited.
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

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/utils"
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

	// Retrieve the module target type variant as set by setVariant
	getTarget() tgtType

	// Get the properties related to which variants are available
	getSplittableProps() *SplittableProps
}

// targetSpecificLibrary extends splittable to allow retrieving specific data
// for host and target.
type targetSpecificLibrary interface {
	splittable

	// Get the target specific properties i.e. host:{} or target:{}
	getTargetSpecific(tgtType) *TargetSpecific

	// Get the set of the module main properties for
	// that target specific properties would be applied to
	targetableProperties() []interface{}
}

// Propagate Host_supported and Target_supported from defaults to
// splittable modules to find out which variations are supported.
func supportedVariantsMutator(mctx blueprint.BottomUpMutatorContext) {

	// No need to do this on defaults modules, as we've flattened the
	// hierarchy
	_, isDefaults := mctx.Module().(*defaults)
	if isDefaults {
		return
	}

	sp, ok := mctx.Module().(splittable)
	if !ok {
		return
	}

	accumulatedProps := SplittableProps{}
	mctx.VisitDirectDeps(func(dep blueprint.Module) {
		if mctx.OtherModuleDependencyTag(dep) == defaultDepTag {
			def, ok := dep.(*defaults)
			if !ok {
				utils.Die("module %s in %s's defaults is not a default",
					dep.Name(), mctx.ModuleName())
			}

			// Append at the same level, so later siblings take precedence
			err := AppendProperties(&accumulatedProps, &def.Properties.SplittableProps)
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					utils.Die("%v", err)
				}
			}
		}
	})

	// Core setting should take precedence over defaults, so prepend
	err := PrependProperties(sp.getSplittableProps(), &accumulatedProps)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			utils.Die("%v", err)
		}
	}
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
