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
	"fmt"
	"reflect"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"github.com/ARM-software/bob-build/internal/utils"
)

// Property concatenation.
//
// Most properties have the behavior that later values override earlier
// values. For example if we passed the compiler "-DVALUE=0
// -DVALUE=1", then the macro VALUE would end up as "1".
//
// A few properties have the opposite behaviour. In particular, since
// include search paths specify a set of directories to look for
// headers, the first directory searched overrides all the others.
//
// Since in Features, Defaults and Targets we copy properties from one
// set to another, we want to be consistent in the way we prepend and
// append arguments so that overrides behave as expected.
//
// Fields in property structs can be tagged with
// `bob:"first_overrides"` to get include search path ordering.
// Otherwise they will get cflag ordering.
//
// The function naming assumes cflag ordering, i.e.
// Append: src cflag properties override dst cflag properties
// Prepend: dst cflag properties override src cflag properties

func orderNormal(property string, dstField, srcField reflect.StructField,
	dstValue, srcValue interface{}) (proptools.Order, error) {
	order := proptools.Append
	if proptools.HasTag(srcField, "bob", "first_overrides") {
		order = proptools.Prepend
	}
	return order, nil
}

func orderReverse(property string, dstField, srcField reflect.StructField,
	dstValue, srcValue interface{}) (proptools.Order, error) {
	order := proptools.Prepend
	if proptools.HasTag(srcField, "bob", "first_overrides") {
		order = proptools.Append
	}
	return order, nil
}

func AppendProperties(dst interface{}, src interface{}) error {
	return proptools.ExtendProperties(dst, src, nil, orderNormal)
}

func AppendMatchingProperties(dst []interface{}, src interface{}) error {
	return proptools.ExtendMatchingProperties(dst, src, nil, orderNormal)
}

func PrependProperties(dst interface{}, src interface{}) error {
	return proptools.ExtendProperties(dst, src, nil, orderReverse)
}

func PrependMatchingProperties(dst []interface{}, src interface{}) error {
	return proptools.ExtendMatchingProperties(dst, src, nil, orderReverse)
}

// Applies default options
func DefaultApplierMutator(mctx blueprint.BottomUpMutatorContext) {
	// The mutator is run bottom up, so modules without dependencies
	// will be processed first.
	//
	// This mutator propagates the properties from the direct default
	// dependencies to the current module.

	// No need to do this on defaults modules, as we've flattened the
	// hierarchy
	_, isDefaults := mctx.Module().(*defaults)
	if isDefaults {
		return
	}

	var defaultableProps []interface{}

	if d, ok := mctx.Module().(defaultable); ok {
		defaultableProps = d.defaultableProperties()
	} else {
		// Not defaultable.
		return
	}

	// Accumulate properties from direct dependencies into an empty defaults
	accumulatedDef := defaults{}
	accumulatedProps := accumulatedDef.defaultableProperties()
	mctx.VisitDirectDeps(func(dep blueprint.Module) {
		if mctx.OtherModuleDependencyTag(dep) == defaultDepTag {
			def, ok := dep.(*defaults)
			if !ok {
				utils.Die("module %s in %s's defaults is not a default",
					dep.Name(), mctx.ModuleName())
			}

			// Append defaults at the same level to maintain cflag order
			err := appendDefaults(accumulatedProps, def.defaultableProperties())
			if err != nil {
				if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
					mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
				} else {
					utils.Die("%s", err)
				}
			}
		}
	})

	// Now apply the defaults to the core module
	// Defaults are more generic, so we prepend to the
	// core module properties.
	//
	// Note: when prepending (pointers to) bools we copy
	// the value if the dst is nil, otherwise the dst
	// value is left alone.
	err := prependDefaults(defaultableProps, accumulatedProps)
	if err != nil {
		if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
			mctx.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
		} else {
			utils.Die("%s", err)
		}
	}
}

func prependDefaults(dst []interface{}, src []interface{}) error {
	// For every property in the destination module (defaultable),
	// we search for the corresponding property within the available
	// set of properties in the source `bob_defaults` module.
	// To prepend them they need to be of the same type.
	for _, defaultableProp := range dst {
		propertyFound := false
		for _, propToApply := range src {
			if reflect.TypeOf(defaultableProp) == reflect.TypeOf(propToApply) {
				err := PrependProperties(defaultableProp, propToApply)

				if err != nil {
					return err
				}

				propertyFound = true
				break
			}
		}

		if !propertyFound {
			return fmt.Errorf("Property of type '%T' was not found in `bob_defaults`", defaultableProp)
		}
	}

	return nil
}

func appendDefaults(dst []interface{}, src []interface{}) error {
	// For every property in the destination module (defaultable),
	// we search for the corresponding property within the available
	// set of properties in the source `bob_defaults` module.
	// To append them they need to be of the same type.
	for _, defaultableProp := range dst {
		propertyFound := false
		for _, propToApply := range src {
			if reflect.TypeOf(defaultableProp) == reflect.TypeOf(propToApply) {
				err := AppendProperties(defaultableProp, propToApply)

				if err != nil {
					return err
				}

				propertyFound = true
				break
			}
		}

		if !propertyFound {
			return fmt.Errorf("Property of type '%T' was not found in `bob_defaults`", defaultableProp)
		}
	}

	return nil
}

// Modules implementing featurable support the use of features and templates.
type featurable interface {
	featurableProperties() []interface{}
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
		props := append([]interface{}{}, m.featurableProperties()...)

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
		var props = []propmap{{m.featurableProperties(), m.features()}}

		// Apply features in target-specific properties.
		// This should happen for all modules which support host:{} and target:{}
		if ts, ok := module.(targetSpecificLibrary); ok {
			host := ts.getTargetSpecific(tgtTypeHost)
			target := ts.getTargetSpecific(tgtTypeTarget)

			var tgtprops = []propmap{
				{[]interface{}{host.getTargetSpecificProps()}, &host.Features},
				{[]interface{}{target.getTargetSpecificProps()}, &target.Features},
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
					utils.Die("%s", err)
				}
			}
		}
	}
}
